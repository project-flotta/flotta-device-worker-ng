package reconcile

import (
	context "context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"go.uber.org/zap"
)

type logger struct {
	nextSyncFunc syncFunc
}

func (l *logger) wrap(s syncFunc) syncFunc {
	l.nextSyncFunc = s
	return l.sync
}

func (l *logger) sync(ctx context.Context, j *entity.Job, executor common.Executor) (entity.JobState, error) {
	oldState := j.CurrentState()
	state, err := l.nextSyncFunc(ctx, j, executor)
	if oldState != state {
		zap.S().Infof("job '%s' changed state from '%s' to '%s'", j.ID(), oldState.String(), state)
	}
	return state, err
}
