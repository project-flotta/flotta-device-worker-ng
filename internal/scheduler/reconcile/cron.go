package reconcile

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type cronHandler struct {
	nextSyncFunc syncFunc
}

func (c *cronHandler) wrap(s syncFunc) syncFunc {
	c.nextSyncFunc = s
	return c.sync
}

func (c *cronHandler) sync(ctx context.Context, j *job.DefaultJob, executor common.Executor) error {
	// check if the job has a cron spec or the target state is running
	if j.Cron() == nil || j.TargetState().OneOf(job.UnknownState, job.ExitedState) {
		return c.nextSyncFunc(ctx, j, executor)
	}

	if time.Now().Before(j.Cron().Next) {
		zap.S().Debugw("cannot reconcile the job yet", "job_id", j.ID(), "next_time", j.Cron().Next)
		return nil
	}

	j.Cron().ComputeNext()
	return c.nextSyncFunc(ctx, j, executor)
}
