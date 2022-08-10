package scheduler

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
	"go.uber.org/zap"
)

type actionType int

const (
	defaultHeartbeatPeriod            = 2 * time.Second
	gracefullShutdown                 = 5 * time.Second
	runAction              actionType = iota
	stopAction
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
					if task.IsEnabled() && (task.CurrentState() == TaskStateRunning || task.CurrentState() == TaskStateDeploying) {
						task.MarkForStop()
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
					// stop the old one
					oldTask.MarkForStop()
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

				if !s.transition(task) {
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

/* transition tries to transition the task to a next state based on the current state.
- if current state is ready then pass to deploying.
- if current state is running and the task has been desactivated than stop it.
- if current state is exited try to restarted
- if current state is stopped than restarted
*/
func (s *Scheduler) transition(t *Task) bool {
	if t.CurrentState() != t.NextState() {
		return true // already mutated
	}
	switch t.CurrentState() {
	case TaskStateReady:
		t.MarkForDeploy()
		return true
	case TaskStateRunning:
		if !t.IsEnabled() {
			t.MarkForStop()
			return true
		}
	case TaskStateStopped:
		fallthrough
	case TaskStateUnknown:
		fallthrough
	case TaskStateExited:
		if !t.CanRun() {
			return false
		}
		t.MarkForDeploy()
		return true
	}
	return false
}
