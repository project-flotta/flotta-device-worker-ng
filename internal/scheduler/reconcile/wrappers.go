package reconcile

import (
	context "context"
	"errors"

	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
	"go.uber.org/zap"
)

type syncFuncWrapper func(s syncFunc) syncFunc

type logWrapper struct {
	syncFunc syncFunc
}

func (l *logWrapper) wrap(s syncFunc) syncFunc {
	l.syncFunc = s
	return l.sync
}

func (l *logWrapper) sync(ctx context.Context, t common.Job, executor common.Executor) error {
	oldState := t.CurrentState()
	err := l.syncFunc(ctx, t, executor)
	if oldState != t.CurrentState() {
		zap.S().Infof("task '%s' changed state from '%s' to '%s'", t.ID(), oldState.String(), t.CurrentState().String())
	}
	return err
}

type retryWrapper struct {
	attempts   int
	maxAttemps int
	syncFunc   syncFunc
}

func (r *retryWrapper) wrap(s syncFunc) syncFunc {
	r.syncFunc = s
	return r.sync
}

func (r *retryWrapper) sync(ctx context.Context, t common.Job, executor common.Executor) error {
	status, err := executor.GetState(context.TODO(), t.Workload())
	if err != nil {
		return err
	}

	state := job.NewState(status)
	t.SetCurrentState(state)

	if r.attempts >= r.maxAttemps {
		zap.S().Errorw("cannot restart task", "task_id", t.ID(), "too many failures", r.attempts)
		return errors.New("failed to restart task. too many failures")
	}

	if t.CurrentState().OneOf(job.ExitedState, job.UnknownState) && t.TargetState() == job.RunningState {
		r.attempts++
		zap.S().Infow("running attempts", "task_id", t.ID(), "attempts", r.attempts)
	}

	err = r.syncFunc(ctx, t, executor)
	return err
}
