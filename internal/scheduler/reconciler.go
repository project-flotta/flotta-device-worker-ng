package scheduler

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type syncTaskFunc func(ctx context.Context, t Task, executor Executor) (State, error)

type reconciler struct {
	syncFuncs map[entity.WorkloadKind]syncTaskFunc
}

func newReconciler() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncTaskFunc),
	}
	r.syncFuncs[entity.PodKind] = createPodmanSyncFunc()
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, tasks []Task, ex Executor) {
	for _, t := range tasks {
		fn, ok := r.syncFuncs[t.Workload().Kind()]
		if !ok {
			zap.S().Error("task kind not supported")
			continue
		}
		fn(ctx, t, ex)
	}
}

func createPodmanSyncFunc() syncTaskFunc {
	return func(ctx context.Context, t Task, executor Executor) (currentState State, err error) {
		zap.S().Debugw("start sync", "task_id", t.ID())
		defer func() {
			zap.S().Debugw("sync done", "task_id", t.ID(), "current_state", currentState.String(), "error", err)
		}()

		status, err := executor.GetState(context.TODO(), t.Workload())
		if err != nil {
			return UnknownState, err
		}

		state := mapToState(status)
		if state == t.TargetState() {
			return t.TargetState(), nil
		}

		zap.S().Infow("new state found", "task_id", t.ID(), "state", state.String())

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
			t.SetCurrentState(mapToState(newState))
			return t.CurrentState(), nil
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
			t.SetCurrentState(mapToState(newState))
			return t.CurrentState(), nil
		}

		if t.TargetState() == RunningState && state.OneOf(UnknownState, ExitedState, DegradedState) {
			currentState, err = runTask(context.TODO())
			return
		}

		if t.TargetState().OneOf(ExitedState, InactiveState) {
			currentState, err = stopTask(context.TODO())
			return
		}

		t.SetCurrentState(state)

		return
	}
}
