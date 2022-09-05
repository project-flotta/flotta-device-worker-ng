package reconcile

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type logger struct {
	nextSyncFunc syncFunc
}

func (l *logger) wrap(s syncFunc) syncFunc {
	l.nextSyncFunc = s
	return l.sync
}

func (l *logger) sync(ctx context.Context, j *job.DefaultJob, executor common.Executor) error {
	oldState := j.CurrentState()
	err := l.nextSyncFunc(ctx, j, executor)
	if oldState != j.CurrentState() {
		zap.S().Infof("job '%s' changed state from '%s' to '%s'", j.ID(), oldState.String(), j.CurrentState().String())
	}
	return err
}
