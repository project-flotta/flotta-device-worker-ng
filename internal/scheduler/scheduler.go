package scheduler

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
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

type syncResult struct {
	State State
	Err   error
}

type syncTaskFunc func(t Task, executor Executor) syncResult

type Scheduler struct {
	// tasks holds all the current tasks
	tasks *containers.Store[Task]
	// executor
	executor Executor
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// syncFuncs holds a map of sync function for each task
	syncFuncs map[Task]syncTaskFunc
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
		tasks:     containers.NewStore[Task](),
		executor:  executor,
		syncFuncs: make(map[Task]syncTaskFunc),
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
		case o := <-input:
			if o.None {
				for _, t := range s.tasks.ToList() {
					t.MarkForDeletion()
				}
				break
			}
			// add tasks
			taskToRemove := substract(s.tasks.ToList(), o.Value)
			newWorkloads := substract(o.Value, s.tasks.ToList())
			for _, w := range newWorkloads {
				t := NewDefaultTask(w.ID(), w)
				s.syncFuncs[t] = s.createSyncFunc(t)
				t.SetTargetState(RunningState)
				s.tasks.Add(t)
			}
			// remove task which are not found in the EdgeWorkload manifest
			for _, t := range taskToRemove {
				t.MarkForDeletion()
			}
		case <-sync:
			for t, syncFunc := range s.syncFuncs {
				syncFunc(t, s.executor)
			}
			// remove all task marked for deletion and which are in unknown state
			for _, t := range s.tasks.ToList() {
				if t.IsMarkedForDeletion() && t.CurrentState() == UnknownState {
					s.removeTask(t)
				}
			}
		case <-heartbeat.C:
			sync <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) removeTask(t Task) {
	zap.S().Infow("task removed", "task_id", t.ID())
	s.tasks.Delete(t)
	delete(s.syncFuncs, t)
}

func (s *Scheduler) createSyncFunc(t Task) syncTaskFunc {
	return func(t Task, executor Executor) (result syncResult) {
		zap.S().Debugw("start sync", "task_id", t.ID())
		defer func() {
			zap.S().Debugw("sync done", "task_id", t.ID(), "state", result.State.String(), "error", result.Err)
		}()

		if t.IsMarkedForDeletion() {
			t.SetTargetState(ExitedState)
		}

		status, err := executor.GetState(context.TODO(), t.Workload())
		if err != nil {
			return syncResult{
				State: UnknownState,
				Err:   err,
			}
		}

		state := mapToState(status)
		if state == t.TargetState() {
			return syncResult{
				State: t.TargetState(),
			}
		}

		zap.S().Infow("new state found", "task_id", t.ID(), "state", state.String())

		runTask := func(ctx context.Context) syncResult {
			zap.S().Infow("run task", "task_id", t.ID())
			exists, err := executor.Exists(ctx, t.Workload())
			if err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			if exists {
				if err := executor.Remove(ctx, t.Workload()); err != nil {
					return syncResult{State: UnknownState, Err: err}
				}
			}
			if err := executor.Run(ctx, t.Workload()); err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			newState, err := executor.GetState(ctx, t.Workload())
			if err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			t.SetCurrentState(mapToState(newState))
			return syncResult{State: t.CurrentState()}
		}

		stopTask := func(ctx context.Context) syncResult {
			zap.S().Infow("stop task", "task_id", t.ID())
			if err := executor.Stop(ctx, t.Workload()); err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			if err := executor.Remove(ctx, t.Workload()); err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			newState, err := executor.GetState(ctx, t.Workload())
			if err != nil {
				return syncResult{State: UnknownState, Err: err}
			}
			t.SetCurrentState(mapToState(newState))
			return syncResult{State: t.CurrentState()}
		}

		if t.TargetState() == RunningState && state.OneOf(UnknownState, ExitedState, DegradedState) {
			result = runTask(context.TODO())
			return
		}

		if t.TargetState().OneOf(ExitedState, InactiveState) {
			result = stopTask(context.TODO())
			return
		}

		t.SetCurrentState(state)
		result = syncResult{State: state}

		return
	}
}

// evaluate evaluates task's profiles based on current device profile.
func (s *Scheduler) evaluate(t Task) bool {
	return true
}
