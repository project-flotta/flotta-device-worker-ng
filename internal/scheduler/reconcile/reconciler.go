package reconcile

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type syncJobFunc func(ctx context.Context, t common.Job, executor common.Executor) error

type reconciler struct {
	syncFuncs map[entity.WorkloadKind]syncJobFunc
}

func New() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncJobFunc),
	}
	r.syncFuncs[entity.PodKind] = createPodmanSyncFunc()
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, jobs []common.Job, ex common.Executor) {
	for _, j := range jobs {
		fn, ok := r.syncFuncs[j.Workload().Kind()]
		if !ok {
			zap.S().Error("job kind not supported")
			continue
		}
		fn(ctx, j, ex)
	}
}

func createPodmanSyncFunc() syncJobFunc {
	return func(ctx context.Context, j common.Job, executor common.Executor) error {
		status, err := executor.GetState(context.TODO(), j.Workload())
		if err != nil {
			return err
		}

		state := job.NewState(status)
		if state == j.TargetState() {
			return nil
		}

		if j.TargetState().OneOf(job.ReadyState, job.ExitedState, job.InactiveState) && state.OneOf(job.ExitedState, job.UnknownState) {
			return nil
		}

		zap.S().Infow("new state found", "job_id", j.ID(), "state", state.String())

		runJob := func(ctx context.Context) (job.State, error) {
			zap.S().Infow("run job", "job_id", j.ID())
			exists, err := executor.Exists(ctx, j.Workload())
			if err != nil {
				return job.UnknownState, err
			}
			if exists {
				if err := executor.Remove(ctx, j.Workload()); err != nil {
					return job.UnknownState, err
				}
			}
			if err := executor.Run(ctx, j.Workload()); err != nil {
				return job.UnknownState, err
			}
			newState, err := executor.GetState(ctx, j.Workload())
			if err != nil {
				return job.UnknownState, err
			}
			j.SetCurrentState(job.NewState(newState))
			return j.CurrentState(), nil
		}

		stopJob := func(ctx context.Context) (job.State, error) {
			zap.S().Infow("stop job", "job_id", j.ID())
			if err := executor.Stop(ctx, j.Workload()); err != nil {
				return job.UnknownState, err
			}
			if err := executor.Remove(ctx, j.Workload()); err != nil {
				return job.UnknownState, err
			}
			newState, err := executor.GetState(ctx, j.Workload())
			if err != nil {
				return job.UnknownState, err
			}
			j.SetCurrentState(job.NewState(newState))
			return j.CurrentState(), nil
		}

		var currentState job.State
		if j.TargetState() == job.RunningState && state.OneOf(job.UnknownState, job.ExitedState, job.DegradedState) {
			currentState, err = runJob(context.TODO())
			if err != nil {
				return err
			}
		}

		if j.TargetState().OneOf(job.ExitedState, job.InactiveState) {
			currentState, err = stopJob(context.TODO())
			if err != nil {
				return err
			}
		}

		j.SetCurrentState(currentState)
		return nil
	}
}
