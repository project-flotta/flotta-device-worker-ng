package scheduler

import (
	context "context"
	"errors"

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

func (l *logWrapper) sync(ctx context.Context, t Task, executor Executor) error {
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

func (r *retryWrapper) sync(ctx context.Context, t Task, executor Executor) error {
	status, err := executor.GetState(context.TODO(), t.Workload())
	if err != nil {
		return err
	}

	state := mapToState(status)
	t.SetCurrentState(state)

	if r.attempts >= r.maxAttemps {
		zap.S().Errorw("cannot restart task", "task_id", t.ID(), "too many failures", r.attempts)
		return errors.New("failed to restart task. too many failures")
	}

	if t.CurrentState().OneOf(ExitedState, UnknownState) && t.TargetState() == RunningState {
		r.attempts++
		zap.S().Infow("running attempts", "task_id", t.ID(), "attempts", r.attempts)
	}

	err = r.syncFunc(ctx, t, executor)
	return err
}
