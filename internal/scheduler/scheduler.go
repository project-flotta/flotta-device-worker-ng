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
	deletion  string = "deletion"
	stop      string = "stop"
	deploy    string = "deploy"
	disabling string = "disable"
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
	go s.run(runCtx, input, profileUpdateCh)
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown run goroutine
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Message, profileCh chan entity.Message) {
	taskCh := make(chan entity.Option[[]entity.Workload], 1)
	heartbeat := time.NewTicker(defaultHeartbeatPeriod)
	doneExecutionCh := make(chan struct{})
	executionInProgress := false

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
		case o := <-taskCh:
			if o.None {
				// stop tasks
				iter := s.tasks.Iter()
				for iter.HasNext() {
					task, _ := iter.Next()
					if !s.isMarked(task, disabling) && (task.CurrentState() == TaskStateRunning || task.CurrentState() == TaskStateDeploying) {
						s.mark(task, stop)
					}
				}
				break
			}
			// add tasks
			for _, w := range o.Value {
				task := NewTask(w.ID(), w)
				if oldTask, found := s.tasks.Find(task.Name); found {
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
		case <-heartbeat.C:
			// wait until all the task had been sent to the executor
			if executionInProgress {
				break
			}

			taskIter := s.tasks.Iter()
			for taskIter.HasNext() {
				task, _ := taskIter.Next()
				fmt.Printf("task %s hash %s current state %s next state %s\n", task.Name, task.Hash(), task.CurrentState().String(), task.NextState().String())

				// check if result is ready for this task
				// if ready try to mutate the task
				future, ok := s.futures[task.Hash()]
				if ok {
					result, _ := future.Poll()
					if result.IsReady() {
						zap.S().Debugw("poll future", "id", task.Hash(), "result", result)
						task.TransitionTo(result.Value)
					}

					// future is resolved when task has either been stopped or exited.
					if future.Resolved() {
						delete(s.futures, task.Hash())
					}
				}

				if task.InTransition {
					continue
				}

				if !s.mutate(task) {
					continue
				}

				switch task.NextState() {
				case TaskStateStopping:
					zap.S().Debugw("stop task", "id", task.Name)
					s.executionQueue.Push(stopAction, task)
				case TaskStateDeploying:
					zap.S().Debugw("deploy task", "id", task.Name)
					s.executionQueue.Push(runAction, task)
				}

				task.InTransition = true

				// clean task marked for deletion
				s.clean()
			}
			if s.executionQueue.Size() > 0 {
				executionInProgress = true
				go s.execute(context.Background(), doneExecutionCh)
			}
		case <-doneExecutionCh:
			executionInProgress = false
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
				task.TransitionTo(TaskStateStopping)
				task.InTransition = false
				s.executor.Stop(context.Background(), task.Workload)
			case runAction:
				task.TransitionTo(TaskStateDeploying)
				task.InTransition = false
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
	Here is all the logic. There are two mutation: locally and based on the external event (i.e. from Executor).
	The local mutation are made by the scheduler which is trying to advance the task towards _running_ state. Also, it tries to restart any failing task (stopped or exited).
	The local mutation are:
		- from ready to deploying meaning that the task will be sent to the executor
		- from running to stopping meaning that the task needs to be stopped because of two reasons: the specs had changed or the task has been removed from EdgeWorkload manifest.
		- from either stopping or exited or unknown to deploying. The natural behavior of the scheduler is to restart jobs but if the job has been marked or desactivated it will be restart.
	All other mutations (i.e. from deploying to running) are event based. Using futures, the executor will send the new state every time the task changed state. Normally, the sequence of state should follow
	the one in Podman.
	The local mutation are made by marking the task for either running or stopping. A job which is marked for deletion will not be restarted and once it stopped it will be removed from the store.
	A task can be enabled or desactivated based on the evaluation of the profiles. If the evaluation of task's profile resolved to false than the task is desactivated and it will be stopped but not removed from the store.
*/
func (s *Scheduler) mutate(t *Task) bool {
	switch t.CurrentState() {
	case TaskStateReady:
		err := t.SetNextState(TaskStateDeploying)
		if err != nil {
			zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
			return false
		}
		return true
	case TaskStateDeployed:
		fallthrough
	case TaskStateRunning:
		if s.isMarked(t, disabling) || s.isMarked(t, deletion) || s.isMarked(t, stop) {
			err := t.SetNextState(TaskStateStopping)
			if err != nil {
				zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
				return false
			}
			return true
		}
	case TaskStateStopped:
		fallthrough
	case TaskStateUnknown:
		fallthrough
	case TaskStateExited:
		if !t.CanRun() || s.isMarked(t, deletion) {
			return false
		}
		err := t.SetNextState(TaskStateDeploying)
		if err != nil {
			zap.S().Errorw("set next state failed", "id", t.Name, "current_state", t.CurrentState(), "next_state", TaskStateDeployed.String())
			return false
		}
		return true
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
