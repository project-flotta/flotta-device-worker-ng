package reconcile

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type syncFunc func(ctx context.Context, t *job.DefaultJob, executor common.Executor) error

type reconciler struct {
	syncFuncs map[entity.WorkloadKind]syncFunc
}

func New() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncFunc),
	}
	retry := &retryHandler{}
	logger := &logger{}
	cron := cronHandler{}
	r.syncFuncs[entity.PodKind] = cron.wrap(retry.wrap(logger.wrap(createPodmanSyncFunc())))
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, jobs []*job.DefaultJob, ex common.Executor) {
	for _, j := range jobs {
		fn, ok := r.syncFuncs[j.Workload().Kind()]
		if !ok {
			zap.S().Error("job kind not supported")
			continue
		}
		fn(ctx, j, ex)
	}
}

func createPodmanSyncFunc() syncFunc {
	return func(ctx context.Context, j *job.DefaultJob, executor common.Executor) error {
		if j.CurrentState() == j.TargetState() {
			return nil
		}

		if j.TargetState().OneOf(job.ReadyState, job.ExitedState, job.InactiveState) && j.CurrentState().OneOf(job.ExitedState, job.UnknownState) {
			return nil
		}

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

		var (
			currentState job.State
			err          error
		)
		if j.TargetState() == job.RunningState && j.CurrentState().OneOf(job.UnknownState, job.ExitedState, job.DegradedState) {
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
