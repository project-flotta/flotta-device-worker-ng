package reconcile

import (
	context "context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"go.uber.org/zap"
)

type syncFunc func(ctx context.Context, job *entity.Job, executor common.Executor) (entity.JobState, error)

type reconciler struct {
	syncFuncs map[entity.WorkloadKind]syncFunc
	futureCh  map[string]chan entity.Result[entity.JobState]
}

func New() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncFunc),
	}
	logger := &logger{}
	r.syncFuncs[entity.PodKind] = logger.wrap(createPodmanSyncFunc())
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, job *entity.Job, ex common.Executor) *entity.Future[entity.Result[entity.JobState]] {
	fn, ok := r.syncFuncs[job.Workload().Kind()]
	if !ok {
		zap.S().Error("job kind not supported")
	}

	ch := make(chan entity.Result[entity.JobState])
	go futureWrapper(ch, func() (entity.JobState, error) {
		return fn(ctx, job, ex)
	})
	return entity.NewFuture(ch)
}

func futureWrapper(ch chan entity.Result[entity.JobState], fn func() (entity.JobState, error)) {
	state, err := fn()
	ch <- entity.Result[entity.JobState]{
		Value: state,
		Error: err,
	}
	close(ch)
}

func createPodmanSyncFunc() syncFunc {
	return func(ctx context.Context, j *entity.Job, executor common.Executor) (entity.JobState, error) {
		if j.CurrentState() == j.TargetState() {
			return j.CurrentState(), nil
		}

		if j.TargetState().OneOf(entity.ReadyState, entity.ExitedState, entity.InactiveState) && j.CurrentState().OneOf(entity.ExitedState, entity.UnknownState) {
			return j.TargetState(), nil
		}

		runJob := func(ctx context.Context) (entity.JobState, error) {
			zap.S().Infow("run job", "job_id", j.ID())
			exists, err := executor.Exists(ctx, j.Workload())
			if err != nil {
				return entity.UnknownState, err
			}
			if exists {
				if err := executor.Remove(ctx, j.Workload()); err != nil {
					return entity.UnknownState, err
				}
			}
			if err := executor.Run(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
			<-time.After(1 * time.Second)
			newState, err := executor.GetState(ctx, j.Workload())
			if err != nil {
				return entity.UnknownState, err
			}
			return newState, nil
		}

		stopJob := func(ctx context.Context) (entity.JobState, error) {
			zap.S().Infow("stop job", "job_id", j.ID())
			if err := executor.Stop(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
			if err := executor.Remove(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
			<-time.After(1 * time.Second)
			newState, err := executor.GetState(ctx, j.Workload())
			if err != nil {
				return entity.UnknownState, err
			}
			return newState, nil
		}

		var (
			currentState entity.JobState
			err          error
		)
		if j.TargetState() == entity.RunningState && j.CurrentState().OneOf(entity.ReadyState, entity.UnknownState, entity.ExitedState, entity.DegradedState) {
			currentState, err = runJob(context.TODO())
		}

		if j.TargetState().OneOf(entity.ExitedState, entity.InactiveState) {
			currentState, err = stopJob(context.TODO())
		}

		return currentState, err
	}
}
