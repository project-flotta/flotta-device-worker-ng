package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	stateMachine "github.com/qmuntal/stateless"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type TaskState int

func (ts TaskState) String() string {
	switch ts {
	case TaskStateReady:
		return "ready"
	case TaskStateDeploying:
		return "deploying"
	case TaskStateDeployed:
		return "deployed"
	case TaskStateRunning:
		return "running"
	case TaskStateStopping:
		return "stopping"
	case TaskStateStopped:
		return "stopped"
	case TaskStateExited:
		return "exited"
	default:
		return "unknown"
	}
}

const (
	// TaskStateReady indicates that the task ready to be deloyed
	TaskStateReady TaskState = iota
	// TaskStateDeploying indicates that the task is currently deploying
	TaskStateDeploying
	// TaskStateDeployed indicates that the task has been deployed but not started yet.
	TaskStateDeployed
	// TaskStateRunning indicates that the task is running
	TaskStateRunning
	// TaskStateStopping indicates that the task is about to be stopped
	TaskStateStopping
	// TaskStateStopped indicates that the task is stopped
	TaskStateStopped
	// TaskStateExited indicates that the task has been stopped with an error
	TaskStateExited
	// TaskStateUnknown indicates that the task is in an unknown state
	TaskStateUnknown

	triggerReady    = "ready"
	triggerDeploy   = "deploy"
	triggerDeployed = "deployed"
	triggerRun      = "run"
	triggerStop     = "stop"
	triggerStopped  = "stopped"
	triggerError    = "error"
	triggerUnknown  = "unknown"
)

type ExecutionEvent struct {
	TaskID string
	State  TaskState
	Error  error
}

type Meta struct {
	marks map[string]string
}

func (m *Meta) SetMark(key, val string) {
	m.marks[key] = val
}

func (m *Meta) GetMark(key string) (value string, ok bool) {
	value, ok = m.marks[key]
	return
}

func (m *Meta) GetMarks() []string {
	marks := make([]string, 0, len(m.marks))
	for k := range m.marks {
		marks = append(marks, k)
	}
	return marks
}

// ADD metadata data to be able to MarkForDeletion MarkForStopping MarkForRunning
type Task struct {
	Meta
	// Name of the task
	Name string
	// workload
	Workload entity.Workload
	// failures counts the number of failures to run the workload
	failures int
	// nextState holds the desired next state of the task
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	nextState TaskState
	// InTransition is set to bool if an action to reconcile current and next state has been taken
	InTransition bool
	// state machine
	machine *stateMachine.StateMachine
}

func NewTask(name string, w entity.Workload) *Task {
	t := Task{
		Meta: Meta{
			marks: make(map[string]string),
		},
		Name:      name,
		Workload:  w,
		nextState: TaskStateReady,
	}

	t.machine = stateMachine.NewStateMachine(TaskStateReady)
	t.machine.Configure(TaskStateReady).
		Permit(triggerDeploy, TaskStateDeploying)

	t.machine.Configure(TaskStateDeploying).
		Permit(triggerDeployed, TaskStateDeployed).
		Permit(triggerError, TaskStateExited)

	t.machine.Configure(TaskStateDeployed).
		Permit(triggerRun, TaskStateRunning).
		Permit(triggerError, TaskStateExited).
		Permit(triggerUnknown, TaskStateUnknown).
		Permit(triggerStop, TaskStateStopping)

	t.machine.Configure(TaskStateRunning).
		Permit(triggerStop, TaskStateStopping).
		Permit(triggerStopped, TaskStateStopped).
		Permit(triggerError, TaskStateExited).
		Permit(triggerUnknown, TaskStateUnknown)

	t.machine.Configure(TaskStateStopping).
		Permit(triggerStopped, TaskStateStopped)

	t.machine.Configure(TaskStateStopped).
		Permit(triggerDeploy, TaskStateDeploying).
		Permit(triggerReady, TaskStateReady)

	t.machine.Configure(TaskStateExited).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			t.failures++
			return nil
		}).
		Permit(triggerDeploy, TaskStateDeploying).
		Permit(triggerReady, TaskStateReady)

	t.machine.Configure(TaskStateUnknown).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			t.failures++
			return nil
		}).
		Permit(triggerDeploy, TaskStateDeploying).
		Permit(triggerReady, TaskStateReady)

	t.machine.OnTransitioned(func(ctx context.Context, tt stateMachine.Transition) {
		fmt.Printf("task %s transitioned from %s to %s\n", t.Name, tt.Source, tt.Destination)
		zap.S().Debugf("task %s transitioned from %s to %s", t.Name, tt.Destination, tt.Source)
	})

	return &t
}

func (t *Task) SetNextState(nextState TaskState) error {
	if t.NextState() == nextState || t.CurrentState() == nextState {
		return nil
	}

	switch nextState {
	case TaskStateDeploying:
		ok, err := t.machine.CanFire(triggerDeploy)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("trask cannot be transitioned to '%s'", nextState.String())
		}
	case TaskStateStopping:
		ok, err := t.machine.CanFire(triggerStop)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("trask cannot be transitioned to '%s'", nextState.String())
		}
	default:
		return fmt.Errorf("task cannot be transitioned to '%s'", nextState.String())
	}

	t.nextState = nextState

	return nil
}

func (t *Task) NextState() TaskState {
	return t.nextState
}

func (t *Task) CurrentState() TaskState {
	return t.machine.MustState().(TaskState)
}

// CanRun returns true if the task can be executed
// TBD what is the conditions when the task cannot be executed anymore?
// After how many retries we are giving up?
func (t *Task) CanRun() bool {
	return t.failures <= 3
}

func (t *Task) TransitionTo(s TaskState) {
	var err error
	switch s {
	case TaskStateDeploying:
		err = t.machine.Fire(triggerDeploy)
	case TaskStateDeployed:
		err = t.machine.Fire(triggerDeployed)
	case TaskStateRunning:
		err = t.machine.Fire(triggerRun)
	case TaskStateStopping:
		err = t.machine.Fire(triggerStop)
	case TaskStateStopped:
		err = t.machine.Fire(triggerStopped)
	case TaskStateExited:
		err = t.machine.Fire(triggerError)
	case TaskStateUnknown:
		err = t.machine.Fire(triggerUnknown)
	}

	if err != nil {
		fmt.Println(err)
		zap.S().Errorw("failed to transition task", "id", t.Name, "error", err)
		return
	}

	// mutate the nextState to the current state and let the scheduler decide what to do next
	t.nextState = t.CurrentState()
	t.InTransition = false
}

func (t *Task) String() string {
	task := struct {
		Name         string `json:"name"`
		Workload     string `json:"workload"`
		CurrentState string `json:"current_state"`
		NextState    string `json:"next_state"`
		Enabled      bool   `json:"enabled"`
	}{
		Name:         t.Name,
		Workload:     t.Workload.String(),
		NextState:    t.NextState().String(),
		CurrentState: t.CurrentState().String(),
	}

	json, err := json.Marshal(task)
	if err != nil {
		return "error marshaling"
	}

	return string(json)
}

func (t *Task) Hash() string {
	return t.Workload.Hash()
}

func (t *Task) ID() string {
	return t.Name
}
