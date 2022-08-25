package scheduler

import (
	"encoding/json"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type Task interface {
	Name() string
	TargetState() State
	CurrentState() State
	SetTargetState(state State) error
	SetCurrentState(state State)
	String() string
	ID() string
	Equal(other Task) bool
	Workload() entity.Workload
	MarkForDeletion()
	IsMarkedForDeletion() bool
}

type Mark struct {
	Kind  string
	Value string
}

type DefaultTask struct {
	// workload
	workload entity.Workload
	// Name of the task
	name string
	// currentState holds the current state of the task
	currentState State
	// nextState holds the desired next state of the task
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	targetState       State
	markedForDeletion bool
}

func NewDefaultTask(name string, w entity.Workload) *DefaultTask {
	return newTask(name, w)
}

func newTask(name string, w entity.Workload) *DefaultTask {
	t := DefaultTask{
		name:         name,
		workload:     w,
		currentState: ReadyState,
	}

	return &t
}

func (t *DefaultTask) SetTargetState(state State) error {
	zap.S().Debugw("new target state", "task_id", t.ID(), "target_state", state)
	t.targetState = state
	return nil
}

func (t *DefaultTask) TargetState() State {
	return t.targetState
}

func (t *DefaultTask) CurrentState() State {
	return t.currentState
}

func (t *DefaultTask) SetCurrentState(currentState State) {
	t.currentState = currentState
}

func (t *DefaultTask) String() string {
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

func (t *DefaultTask) Equal(other Task) bool {
	return t.workload.Hash() == other.Workload().Hash()
}

func (t *DefaultTask) ID() string {
	return t.workload.Hash()[:8]
}

func (t *DefaultTask) Name() string {
	return t.name
}

func (t *DefaultTask) Workload() entity.Workload {
	return t.workload
}

func (t *DefaultTask) MarkForDeletion() {
	t.markedForDeletion = true
}

func (t *DefaultTask) IsMarkedForDeletion() bool {
	return t.markedForDeletion
}
