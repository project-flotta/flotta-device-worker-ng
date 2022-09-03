package reconcile

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"go.uber.org/zap"
)

type logWrapper struct {
	nextSyncFunc syncFunc
}

func (l *logWrapper) wrap(s syncFunc) syncFunc {
	l.nextSyncFunc = s
	return l.sync
}

func (l *logWrapper) sync(ctx context.Context, j common.Job, executor common.Executor) error {
	oldState := j.CurrentState()
	err := l.nextSyncFunc(ctx, j, executor)
	if oldState != j.CurrentState() {
		zap.S().Infof("job '%s' changed state from '%s' to '%s'", j.ID(), oldState.String(), j.CurrentState().String())
	}
	return err
}
