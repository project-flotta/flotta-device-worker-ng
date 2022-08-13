package scheduler

import (
	"errors"

	"go.uber.org/zap"
)

const (
	// marks type
	deletion   string = "deletion"
	stop       string = "stop"
	deploy     string = "deploy"
	inactive   string = "inactive"
	mutateMark string = "mutate"
)

// RestartGuard is the function which return true if the task can be restarted.
// It is used to check if failing task can be restarted or not.
type RestartGuard func(t *Task) bool

type mutator struct {
	canRestart RestartGuard
}

func NewMutator() *mutator {
	return &mutator{
		// standard guard for failing task. Return false if failures > 3
		canRestart: func(t *Task) bool {
			return t.Failures <= 3
		},
	}
}

func NewMutatorWithRestartGuard(r RestartGuard) *mutator {
	return &mutator{r}
}

/*
	Mutate tries to find what is the next state based on marks and/or current state.
*/
func (m *mutator) Mutate(t *Task) (bool, error) {
	// process task with marks
	for _, mark := range t.GetMarks() {
		switch mark {
		case mutateMark:
			val, ok := t.GetMark(mark)
			if !ok {
				zap.S().Warnw("mutation value not found", "task_id", t.ID(), "mark", mark)
				return false, errors.New("mutation value not found")
			}
			// if the task is exited or in unknown state and cannot be restarted.
			if t.CurrentState().OneOf(TaskStateExited, TaskStateUnknown) && !m.canRestart(t) {
				t.SetNextState(t.CurrentState())
				return false, nil
			}
			t.SetNextState(val.(TaskState))
			t.RemoveMark(mutateMark)
			return true, nil
		case stop:
			if !t.CurrentState().OneOf(TaskStateDeploying, TaskStateDeployed, TaskStateRunning) {
				return false, nil
			}
			t.SetNextState(TaskStateStopping)
			t.RemoveMark(stop)
			return true, nil
		case inactive:
			if !t.CurrentState().OneOf(TaskStateReady, TaskStateStopped, TaskStateExited, TaskStateUnknown) {
				return false, nil
			}
			// transition to inactive is permitted only from ready, stopped, exit or unknown state.
			// a running job must be stopped before make it transition to inactive
			t.SetNextState(TaskStateInactive)
			return true, nil
		}
	}
	return false, nil
}
