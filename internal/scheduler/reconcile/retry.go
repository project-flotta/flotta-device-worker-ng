package reconcile

import (
	context "context"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type jobRetrier struct {
	nextRetry  time.Time
	expBackoff backoff.BackOff
}

func (jr *jobRetrier) ComputeNextRetryTime() {
	jr.nextRetry = jr.nextRetry.Add(jr.expBackoff.NextBackOff())
}

func newRetryWrapper() *retryWrapper {
	return &retryWrapper{
		jobRetriers: make(map[string]*jobRetrier),
	}
}

type retryWrapper struct {
	jobRetriers map[string]*jobRetrier
	// nextSyncFunc holds the next sync function to be called
	nextSyncFunc syncFunc
}

func (r *retryWrapper) createRetrier(jobID string) *jobRetrier {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 30 * time.Second
	exp.MaxInterval = 10 * time.Minute
	exp.Multiplier = 3
	jr := &jobRetrier{
		nextRetry:  time.Now(),
		expBackoff: exp,
	}
	zap.S().Infow("backoff retry created", "job_id", jobID)
	r.jobRetriers[jobID] = jr
	return jr
}

func (r *retryWrapper) wrap(s syncFunc) syncFunc {
	r.nextSyncFunc = s
	return r.sync
}

func (r *retryWrapper) sync(ctx context.Context, j common.Job, executor common.Executor) error {
	status, err := executor.GetState(context.TODO(), j.Workload())
	if err != nil {
		return err
	}

	state := job.NewState(status)
	j.SetCurrentState(state)

	// get or create the job retry
	var jr *jobRetrier
	if i, found := r.jobRetriers[j.ID()]; !found {
		jr = r.createRetrier(j.ID())
	} else {
		jr = i
	}

	if j.CurrentState() != j.TargetState() && jr.nextRetry.After(time.Now()) {
		zap.S().Debugf("cannot reconcile the job yet. Wait until '%s'", jr.nextRetry)
		return nil
	}

	if j.CurrentState().OneOf(job.ExitedState, job.UnknownState) && j.TargetState() == job.RunningState {
		jr.ComputeNextRetryTime()
		zap.S().Infow("job restarted", "job_id", j.ID(), "next retry after", jr.nextRetry)
	}

	err = r.nextSyncFunc(ctx, j, executor)
	return err
}
