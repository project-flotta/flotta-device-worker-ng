package task

import (
	"encoding/json"
	"fmt"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
)

type Task interface {
	Name() string
	NextState() State
	CurrentState() State
	SetNextState(state State) error
	SetCurrentState(state State)
	String() string
	ID() string
	Equal(other Task) bool
	Workload() entity.Workload
	HasMarks() bool
	AddMark(mark, value string)
	Peek() Mark
	PopMark() Mark
	FindState(nextState State) (Paths, error)
}

type Mark struct {
	Kind  string
	Value string
}

type DefaultTask struct {
	// workload
	workload entity.Workload
	// marks holds the marks
	marks *containers.Queue[Mark]
	// Name of the task
	name string
	// blueprint holds the blueprint of the task
	blueprint *blueprint
	// currentState holds the current state of the task
	currentState State
	// nextState holds the desired next state of the task
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	nextState State
}

func NewDefaultTask(name string, w entity.Workload) *DefaultTask {
	return newTask(name, newPodmanBlueprint(), w)
}

func newTask(name string, bp *blueprint, w entity.Workload) *DefaultTask {
	t := DefaultTask{
		name:      name,
		blueprint: bp,
		workload:  w,
		nextState: ReadyState,
		marks:     containers.NewQueue[Mark](),
	}

	return &t
}

func (t *DefaultTask) SetNextState(nextState State) error {
	fmt.Printf("task '%s' mutate from %s to %s\n", t.ID(), t.currentState.String(), nextState.String())
	t.nextState = nextState
	return nil
}

func (t *DefaultTask) NextState() State {
	return t.nextState
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
		Kind         string `json:"kind"`
		Workload     string `json:"workload"`
		CurrentState string `json:"current_state"`
		NextState    string `json:"next_state"`
	}{
		Name:         t.name,
		Kind:         t.blueprint.Kind.String(),
		Workload:     t.workload.String(),
		CurrentState: t.CurrentState().String(),
		NextState:    t.NextState().String(),
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
	return t.workload.Hash()
}

func (t *DefaultTask) Name() string {
	return t.name
}

func (t *DefaultTask) Kind() string {
	return t.blueprint.Kind.String()
}

func (t *DefaultTask) Workload() entity.Workload {
	return t.workload
}

func (t *DefaultTask) AddMark(mark, value string) {
	t.marks.Push(Mark{mark, value})
}

func (t *DefaultTask) HasMarks() bool {
	return t.marks.Size() > 0
}

func (t *DefaultTask) PopMark() Mark {
	return t.marks.Pop()
}

func (t *DefaultTask) Peek() Mark {
	return t.marks.Peek()
}

func (t *DefaultTask) FindState(nextState State) (Paths, error) {
	return t.blueprint.FindPath(t.currentState, nextState)
}
