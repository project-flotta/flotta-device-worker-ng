package scheduler

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type syncFunc func(ctx context.Context, t Task, executor Executor) error

type reconciler struct {
	syncFuncs map[entity.WorkloadKind]syncFunc
	wrappers  []syncFuncWrapper
}

func newReconciler() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncFunc),
	}
	retryWrapper := &retryWrapper{maxAttemps: 3}
	logger := &logWrapper{}
	r.syncFuncs[entity.PodKind] = retryWrapper.wrap(logger.wrap(createPodmanSyncFunc()))
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, tasks []Task, ex Executor) {
	for _, t := range tasks {
		syncFn, ok := r.syncFuncs[t.Workload().Kind()]
		if !ok {
			zap.S().Error("task kind not supported")
			continue
		}
		syncFn(ctx, t, ex)
	}
}

func createPodmanSyncFunc() syncFunc {
	return func(ctx context.Context, t Task, executor Executor) error {
		if t.CurrentState() == t.TargetState() {
			return nil
		}

		if t.TargetState().OneOf(ReadyState, ExitedState, InactiveState) && t.CurrentState().OneOf(ExitedState, UnknownState) {
			return nil
		}

		runTask := func(ctx context.Context) (State, error) {
			zap.S().Infow("run task", "task_id", t.ID())
			exists, err := executor.Exists(ctx, t.Workload())
			if err != nil {
				return UnknownState, err
			}
			if exists {
				if err := executor.Remove(ctx, t.Workload()); err != nil {
					return UnknownState, err
				}
			}
			if err := executor.Run(ctx, t.Workload()); err != nil {
				return UnknownState, err
			}
			newState, err := executor.GetState(ctx, t.Workload())
			if err != nil {
				return UnknownState, err
			}
			return mapToState(newState), err
		}

		stopTask := func(ctx context.Context) (State, error) {
			zap.S().Infow("stop task", "task_id", t.ID())
			if err := executor.Stop(ctx, t.Workload()); err != nil {
				return UnknownState, err
			}
			if err := executor.Remove(ctx, t.Workload()); err != nil {
				return UnknownState, err
			}
			newState, err := executor.GetState(ctx, t.Workload())
			if err != nil {
				return UnknownState, err
			}
			return mapToState(newState), nil
		}

		if t.TargetState() == RunningState && t.CurrentState().OneOf(UnknownState, ExitedState, DegradedState) {
			currentState, err := runTask(context.TODO())
			if err != nil {
				return err
			}
			t.SetCurrentState(currentState)
		}

		if t.TargetState().OneOf(ExitedState, InactiveState) {
			currentState, err := stopTask(context.TODO())
			if err != nil {
				return err
			}
			t.SetCurrentState(currentState)
		}

		return nil
	}
}
