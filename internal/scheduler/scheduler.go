package scheduler

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

const (
	defaultHeartbeatPeriod = 1 * time.Second
	gracefullShutdown      = 5 * time.Second
)

//go:generate mockgen -package=scheduler -destination=mock_executor.go --build_flags=--mod=mod . Executor
type Executor interface {
	Remove(ctx context.Context, w entity.Workload) error
	Run(ctx context.Context, w entity.Workload) error
	Stop(ctx context.Context, w entity.Workload) error
	GetState(ctx context.Context, w entity.Workload) (string, error)
	Exists(ctx context.Context, w entity.Workload) (bool, error)
}

type Reconciler interface {
	Reconcile(ctx context.Context, tasks []Task, executor Executor)
}

type Evaluator interface {
	Evaluate(ctx context.Context, t Task) bool
}

type Scheduler struct {
	// tasks holds all the current tasks
	tasks *Store[Task]
	// executor
	executor Executor
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// reconciler
	reconciler Reconciler
	// evaluator evaluates each task based on device's profiles
	evaluator Evaluator
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
		tasks:      NewStore[Task](),
		executor:   executor,
		reconciler: newReconciler(),
	}
}

func (s *Scheduler) Start(ctx context.Context, input chan entity.Message, profileUpdateCh chan entity.Message) {
	runCtx, cancel := context.WithCancel(ctx)
	s.runCancel = cancel

	taskCh := make(chan entity.Option[[]entity.Workload])
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
	go s.run(runCtx, taskCh, profileUpdateCh)
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown goroutines
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Option[[]entity.Workload], profileCh chan entity.Message) {
	sync := make(chan struct{}, 1)

	heartbeat := time.NewTicker(defaultHeartbeatPeriod)

	for {
		select {
		case opt := <-input:
			if opt.None {
				for _, t := range s.tasks.ToList() {
					t.MarkForDeletion()
					t.SetTargetState(ExitedState)
				}
				break
			}
			// add tasks
			taskToRemove := substract(s.tasks.ToList(), opt.Value)
			newWorkloads := substract(opt.Value, s.tasks.ToList())
			for _, w := range newWorkloads {
				t := NewDefaultTask(w.ID(), w)
				t.SetTargetState(RunningState)
				s.tasks.Add(t)
			}
			// remove task which are not found in the EdgeWorkload manifest
			for _, t := range taskToRemove {
				t.MarkForDeletion()
				t.SetTargetState(ExitedState)
			}
		case <-sync:
			// reconcile the tasks
			s.reconciler.Reconcile(context.Background(), s.tasks.Clone().ToList(), s.executor)
			// remove all task marked for deletion and which are in unknown state
			for _, t := range s.tasks.ToList() {
				if t.IsMarkedForDeletion() && t.CurrentState() == UnknownState {
					zap.S().Infow("task removed", "task_id", t.ID())
					s.tasks.Delete(t)
				}
			}
		case profiles := <-profileCh:
			zap.S().Debug(profiles)
		case <-heartbeat.C:
			sync <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}
