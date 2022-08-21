package task

import (
	"errors"
	"fmt"

	"github.com/tupyy/device-worker-ng/internal/entity"
)

// wrapper for Task
type retry struct {
	failures    int
	maxFailures int
	t           Task
}

var (
	ErrTooManyFailures = errors.New("too many failures")
)

func NewTaskWithRetry(t Task, maxFailures int) Task {
	return retry{
		t:           t,
		maxFailures: maxFailures,
	}
}

func (r retry) SetNextState(state State) error {
	if state.OneOf(ReadyState, DeployingState) && r.failures > r.maxFailures {
		return fmt.Errorf("%w task_name '%s'", ErrTooManyFailures, r.t.Name())
	}
	return r.t.SetNextState(state)
}

func (r retry) SetCurrentState(state State) {
	if state.OneOf(ErrorState, ExitedState) {
		r.failures++
	}
	r.t.SetCurrentState(state)
}

func (r retry) Name() string {
	return r.t.Name()
}

func (r retry) NextState() State {
	return r.t.NextState()
}

func (r retry) CurrentState() State {
	return r.t.CurrentState()
}

func (r retry) String() string {
	return r.t.String()
}

func (r retry) ID() string {
	return r.t.ID()
}

func (r retry) Equal(other Task) bool {
	return r.t.Equal(other)
}

func (r retry) Workload() entity.Workload {
	return r.t.Workload()
}

func (r retry) HasMarks() bool {
	return r.t.HasMarks()
}

func (r retry) AddMark(mark, value string) {
	r.t.AddMark(mark, value)
}

func (r retry) Peek() Mark {
	return r.t.Peek()
}

func (r retry) PopMark() Mark {
	return r.t.PopMark()
}

func (r retry) FindState(nextState State) (Paths, error) {
	return r.t.FindState(nextState)
}
