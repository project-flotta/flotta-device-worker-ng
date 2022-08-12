package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
	"go.uber.org/zap"
)

type actionType int

const (
	defaultHeartbeatPeriod = 2 * time.Second
	gracefullShutdown      = 5 * time.Second

	// action type
	runAction actionType = iota
	stopAction

	// marks type
	deletion string = "deletion"
	stop     string = "stop"
	deploy   string = "deploy"
	inactive string = "inactive"
)

//go:generate mockgen -package=scheduler -destination=mock_executor.go --build_flags=--mod=mod . Executor
type Executor interface {
	Run(ctx context.Context, w entity.Workload) *Future[TaskState]
	Stop(ctx context.Context, w entity.Workload)
}

type Scheduler struct {
	// tasks holds all the current tasks
	tasks *containers.Store[*Task]
	// futures holds the futures of executed tasks
	// the hash of the task is the key
	futures map[string]*Future[TaskState]
	// executor
	executor Executor
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// executionQueue holds the tasks which must be executed by executor
	executionQueue *containers.Queue[actionType, *Task]
}

// New creates a new scheduler with the default heartbeat period of 2 seconds.
func New(executor Executor) *Scheduler {
	return newExecutor(executor, defaultHeartbeatPeriod)
}

// New creates a new scheduler with the hearbeat period provided by the user.
func NewWitHeartbeatPeriod(executor Executor, heartbeatPeriod time.Duration) *Scheduler {
	return newExecutor(executor, heartbeatPeriod)
}

func newExecutor(executor Executor, heartbeatPeriod time.Duration) *Scheduler {
	return &Scheduler{
		tasks:          containers.NewStore[*Task](),
		futures:        make(map[string]*Future[TaskState]),
		executionQueue: containers.NewQueue[actionType, *Task](),
		executor:       executor,
	}
}

func (s *Scheduler) Start(ctx context.Context, input chan entity.Message, profileUpdateCh chan entity.Message) {
	runCtx, cancel := context.WithCancel(ctx)
	s.runCancel = cancel

	taskCh := make(chan entity.Option[[]entity.Workload])
	go s.run(runCtx, taskCh, profileUpdateCh)
	go func(ctx context.Context) {
		for {
			select {
			case message := <-input:
				switch message.Kind {
				case entity.WorkloadConfigurationMessage:
					val, ok := message.Payload.(entity.Option[[]entity.Workload])
					if !ok {
						zap.S().Errorf("mismatch message payload type. expected workload. got %v", message)
					}
					taskCh <- val
				}
			case <-ctx.Done():
				return
			}
		}
	}(runCtx)
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown goroutines
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Option[[]entity.Workload], profileCh chan entity.Message) {
	execution := make(chan struct{}, 1)
	futures := make(chan struct{}, 1)
	doneExecutionCh := make(chan struct{})
	mutate := make(chan struct{}, 1)

	heartbeat := time.NewTicker(defaultHeartbeatPeriod)

	for {
		select {
		case o := <-input:
			if o.None {
				// stop tasks
				iter := s.tasks.Iter()
				for iter.HasNext() {
					task, _ := iter.Next()
					if !s.isMarked(task, inactive) && (task.CurrentState() == TaskStateRunning || task.CurrentState() == TaskStateDeploying) {
						s.mark(task, stop)
					}
				}
				break
			}
			// add tasks
			for _, w := range o.Value {
				task := NewTask(w.ID(), w)
				if oldTask, found := s.tasks.FindByName(task.Name()); found {
					if oldTask.Hash() == task.Hash() {
						continue
					}
					// something changed in the workload. Stop the old one and start the new one
					zap.S().Infow("workload changed", "name", oldTask.Name)
					// stop the old one and remove it from store
					s.mark(oldTask, stop)
					s.mark(oldTask, deletion)
				}
				s.tasks.Add(task)
			}
		case <-futures:
			// Poll futures.
			for id, future := range s.futures {
				result, _ := future.Poll()
				if result.IsReady() {
					task, ok := s.tasks.FindByID(id)
					if !ok {
						delete(s.futures, id)
					}
					zap.S().Debugw("poll future", "id", task.Hash(), "result", result)
					task.MutateTo(result.Value)
				}

				// future is resolved when task has either been stopped or exited.
				if future.Resolved() {
					delete(s.futures, id)
				}
			}
			mutate <- struct{}{}
		case <-mutate:
			taskIter := s.tasks.Iter()
			for taskIter.HasNext() {
				task, _ := taskIter.Next()
				fmt.Printf("task %s hash %s current state %s next state %s\n", task.Name(), task.Hash(), task.CurrentState().String(), task.NextState().String())

				// evaluate the task
				if !s.evaluate(task) {
					s.mark(task, stop)
					s.mark(task, inactive)
				}

				if !s.transitionToNextState(task) {
					continue
				}

				switch task.NextState() {
				case TaskStateInactive:
					task.MutateTo(TaskStateInactive)
				case TaskStateStopping:
					zap.S().Debugw("stop task", "id", task.Name)
					s.executionQueue.Push(stopAction, task)
				case TaskStateDeploying:
					zap.S().Debugw("deploy task", "id", task.Name)
					s.executionQueue.Push(runAction, task)
				}
			}
			if s.tasks.Len() > 0 {
				execution <- struct{}{}
			}
		case <-execution:
			// clean task marked for deletion
			s.clean()
			// execute every task in the execution queue
			go s.execute(context.Background(), doneExecutionCh)
			// stop heartbeat while we are consuming the execution queue.
			// Once is done, reset the timer.
			heartbeat.Stop()
		case <-doneExecutionCh:
			heartbeat.Reset(defaultHeartbeatPeriod)
		case <-heartbeat.C:
			futures <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, doneCh chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	for s.executionQueue.Size() > 0 {
		select {
		case <-ticker.C:
			// stopping task has high priority
			s.executionQueue.Sort(stopAction)
			action, task, err := s.executionQueue.Pop()
			if err != nil {
				zap.S().Errorw("failed to pop task from queue", "error", err)
				break
			}
			switch action {
			case stopAction:
				task.MutateTo(TaskStateStopping)
				s.executor.Stop(context.Background(), task.Workload)
			case runAction:
				task.MutateTo(TaskStateDeploying)
				future := s.executor.Run(context.Background(), task.Workload)
				s.futures[task.Hash()] = future
			}
		case <-ctx.Done():
			doneCh <- struct{}{}
			return
		}
	}
	doneCh <- struct{}{}
}

func (s *Scheduler) evaluate(t *Task) bool {
	return true
}

/*
	Here is all the logic. There are two mutations: natural and event-based.
	Naturally, the scheduler tries to advance the task towards _running_ state and restart any failing task (stopped, exit or in unknown state). Marks change this behavior.
	The natural mutations are:
		- from ready to deploying meaning that the task will be sent to the executor
		- from either stopping or exited or unknown to deploying. The natural behavior of the scheduler is to restart jobs but this can be changed if the job has been marked.

	If the tasked has been removed from EdgeWorkload or the specs had changed,the scheduler tries to stopped it.
	All other mutations (i.e. from deploying to running) are event based. Using futures, the executor will send the new state every time the task changed state. Normally, the sequence of state should follow
	the one in Podman.

	Marks can change the behavior of task. If a task is keep failing and it passed the failing threshold (TBD), the task is marked as inactive and if the task is currently running it will be stopped and transition into Inactive state afterwards.
*/
func (s *Scheduler) transitionToNextState(t *Task) bool {
	// natural mutation without marks are:
	// from ready to deploying
	// from stopped | unknown | exit to deploying
	if !t.HasMarks() {
		switch t.CurrentState() {
		case TaskStateReady:
			err := t.SetNextState(TaskStateDeploying)
			if err != nil {
				zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
				return false
			}
			return true
		case TaskStateStopped:
			fallthrough
		case TaskStateUnknown:
			fallthrough
		case TaskStateExited:
			// if task cannot be restarted marked inactive
			if !t.CanRun() {
				return false
			}
			err := t.SetNextState(TaskStateDeploying)
			if err != nil {
				zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
				return false
			}
			return true
		}
	}

	// process task with marks
	for _, mark := range t.GetMarks() {
		switch mark {
		case stop:
			if t.CurrentState() != TaskStateDeployed && t.CurrentState() != TaskStateDeploying && t.CurrentState() != TaskStateRunning {
				return false
			}
			err := t.SetNextState(TaskStateStopping)
			if err != nil {
				zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
				return false
			}
			t.RemoveMark(stop)
			return true
		case inactive:
			if t.CurrentState() != TaskStateReady && t.CurrentState() != TaskStateStopped && t.CurrentState() != TaskStateExited && t.CurrentState() != TaskStateUnknown {
				return false
			}
			// transition to inactive is permitted only from ready, stopped, exit or unknown state.
			// a running job must be stopped before make it transition to inactive
			err := t.SetNextState(TaskStateInactive)
			if err != nil {
				zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
				return false
			}
			return true

		}
	}

	return false
}

// remove task marked for deletion and which are stopped, exited or unknown
func (s *Scheduler) clean() {
	for {
		dirty := false
		for i := 0; i < s.tasks.Len(); i++ {
			if t, ok := s.tasks.Get(i); ok && s.isMarked(t, deletion) && (t.CurrentState() == TaskStateStopped || t.CurrentState() == TaskStateExited || t.CurrentState() == TaskStateUnknown) {
				s.tasks.Delete(t)
				dirty = true
				break
			}
		}
		if !dirty {
			break
		}
	}
}

func (s *Scheduler) mark(t *Task, mark string) {
	t.SetMark(mark, mark)
}

func (s *Scheduler) isMarked(t *Task, mark string) bool {
	_, marked := t.GetMark(mark)
	return marked
}
