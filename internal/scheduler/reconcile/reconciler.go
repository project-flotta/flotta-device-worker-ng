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
}

func New() *reconciler {
	r := &reconciler{
		syncFuncs: make(map[entity.WorkloadKind]syncFunc),
	}
	r.syncFuncs[entity.PodKind] = createPodmanSyncFunc()
	r.syncFuncs[entity.K8SKind] = createK8SSyncFunc()
	return r
}

func (r *reconciler) Reconcile(ctx context.Context, job *entity.Job, ex common.Executor) *entity.Future[entity.Result[entity.JobState]] {
	zap.S().Debugw("reconcile started", "now", time.Now())
	fn, ok := r.syncFuncs[job.Workload().Kind()]
	if !ok {
		zap.S().Error("job kind not supported")
	}

	ch := make(chan entity.Result[entity.JobState])
	go futureWrapper(ctx, ch, job, ex, fn)
	return entity.NewFuture(ch)
}

func futureWrapper(ctx context.Context, ch chan entity.Result[entity.JobState], job *entity.Job, ex common.Executor, fn syncFunc) {
	state, err := fn(ctx, job, ex)
	ch <- entity.Result[entity.JobState]{
		Value: state,
		Error: err,
	}
	close(ch)
}

func createPodmanSyncFunc() syncFunc {
	fn := func(ctx context.Context, j *entity.Job, executor common.Executor) (state entity.JobState, err error) {
		if j.CurrentState() == j.TargetState() {
			return j.CurrentState(), nil
		}

		if j.TargetState() == entity.RunningState {
			zap.S().Infow("run job", "job_id", j.ID())
			exists, errE := executor.Exists(ctx, j.Workload())
			if errE != nil {
				return entity.UnknownState, errE
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
			state, err = executor.GetState(ctx, j.Workload())
			if err != nil {
				return entity.UnknownState, err
			}
			return
		}

		if j.TargetState().OneOf(entity.ExitedState, entity.InactiveState) {
			zap.S().Infow("stop job", "job_id", j.ID())
			if err := executor.Stop(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
			if err := executor.Remove(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
			<-time.After(1 * time.Second)
			state, err = executor.GetState(ctx, j.Workload())
			if err != nil {
				return entity.UnknownState, err
			}
			return
		}

		return j.CurrentState(), nil
	}
	logger := &logger{}
	return logger.wrap(fn)
}

func createK8SSyncFunc() syncFunc {
	fn := func(ctx context.Context, j *entity.Job, executor common.Executor) (state entity.JobState, err error) {
		if j.CurrentState() == j.TargetState() {
			return j.CurrentState(), nil
		}

		if j.TargetState() == entity.RunningState {
			zap.S().Infow("create deployment", "job_id", j.ID())
			if err := executor.Run(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
		}

		if j.TargetState().OneOf(entity.ExitedState, entity.InactiveState, entity.UnknownState) {
			zap.S().Infow("remove deployment from k8s", "job_id", j.ID())
			if err := executor.Remove(ctx, j.Workload()); err != nil {
				return entity.UnknownState, err
			}
		}
		<-time.After(1 * time.Second)
		state, err = executor.GetState(ctx, j.Workload())
		if err != nil {
			return entity.UnknownState, err
		}
		return state, nil
	}
	logger := &logger{}
	return logger.wrap(fn)
}
