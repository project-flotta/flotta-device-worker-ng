package job

import (
	"encoding/json"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type DefaultJob struct {
	// workload
	workload entity.Workload
	// Name of the task
	name string
	// currentState holds the current state of the task
	currentState State
	// targetState holds the desired next state of the task
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	targetState       State
	markedForDeletion bool
}

func NewDefaultJob(name string, w entity.Workload) *DefaultJob {
	return newTask(name, w)
}

func newTask(name string, w entity.Workload) *DefaultJob {
	t := DefaultJob{
		name:         name,
		workload:     w,
		currentState: ReadyState,
		targetState:  ReadyState,
	}

	return &t
}

func (t *DefaultJob) SetTargetState(state State) error {
	zap.S().Debugw("set target state", "task_id", t.ID(), "target_state", state)
	t.targetState = state
	return nil
}

func (t *DefaultJob) TargetState() State {
	return t.targetState
}

func (t *DefaultJob) CurrentState() State {
	return t.currentState
}

func (t *DefaultJob) SetCurrentState(currentState State) {
	t.currentState = currentState
}

func (t *DefaultJob) String() string {
	task := struct {
		Name         string `json:"name"`
		Workload     string `json:"workload"`
		CurrentState string `json:"current_state"`
		TargetState  string `json:"target_state"`
	}{
		Name:         t.name,
		Workload:     t.workload.String(),
		CurrentState: t.CurrentState().String(),
		TargetState:  t.TargetState().String(),
	}

	json, err := json.Marshal(task)
	if err != nil {
		return "error marshaling"
	}

	return string(json)
}

func (t *DefaultJob) ID() string {
	return t.workload.ID()
}

func (t *DefaultJob) Name() string {
	return t.name
}

func (t *DefaultJob) Workload() entity.Workload {
	return t.workload
}

func (t *DefaultJob) MarkForDeletion() {
	t.markedForDeletion = true
}

func (t *DefaultJob) IsMarkedForDeletion() bool {
	return t.markedForDeletion
}
