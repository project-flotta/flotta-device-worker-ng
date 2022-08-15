package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qmuntal/stateless"
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
	case TaskStateDegraded:
		return "degraded"
	case TaskStateStopped:
		return "stopped"
	case TaskStateExited:
		return "exited"
	case TaskStateError:
		return "error"
	case TaskStateInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

func (ts TaskState) OneOf(states ...TaskState) bool {
	for _, s := range states {
		if ts == s {
			return true
		}
	}
	return false
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
	// TaskStateStopped indicates that the task has been stopped without error
	TaskStateStopped
	// TaskStateDegraded indicates that the task is an degrated state like a pod with containers stopped.
	TaskStateDegraded
	// TaskStateExited indicates that the task has been stopped with an error
	TaskStateExited
	// TaskStateError indicates that deploying of the task has resulted in error.
	TaskStateError
	// TaskStateUnknown indicates that the task is in an unknown state
	TaskStateUnknown
	// TaskStateInactive indicates that the task is in an inactive state.
	TaskStateInactive
	// TaskStateDeletion indicates that the task is being removed from the scheduler.
	TaskStateDeletion

	triggerReady    = "ready"
	triggerDeploy   = "deploy"
	triggerDeployed = "deployed"
	triggerRun      = "run"
	triggerStop     = "stop"
	triggerStopped  = "stopped"
	triggerExit     = "exit"
	tiggerDegraded  = "degraded"
	triggerInactive = "inactive"
	triggerError    = "error"
	triggerUnknown  = "unknown"
	triggerDegraded = "degraded"
)

type ExecutionEvent struct {
	TaskID string
	State  TaskState
	Error  error
}

type Meta struct {
}

// ADD metadata data to be able to MarkForDeletion MarkForStopping MarkForRunning
type Task struct {
	Meta
	// workload
	Workload entity.Workload
	// Failures counts the number of Failures to run the workload
	Failures int
	// Name of the task
	name string
	// nextState holds the desired next state of the task
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	nextState TaskState
	// state machine
	machine *stateless.StateMachine
	// marks holds the marks
	marks map[string]interface{}
}

func NewTask(name string, w entity.Workload) *Task {
	return _new(name, w)
}

func _new(name string, w entity.Workload) *Task {
	t := Task{
		Meta:      Meta{},
		name:      name,
		Workload:  w,
		nextState: TaskStateReady,
		marks:     make(map[string]interface{}),
	}

	t.machine = t.initStateless()

	return &t
}

func (t *Task) SetNextState(nextState TaskState) {
	t.nextState = nextState
}

func (t *Task) NextState() TaskState {
	return t.nextState
}

func (t *Task) CurrentState() TaskState {
	return t.machine.MustState().(TaskState)
}

func (t *Task) Reset() {
	t.Failures = 0
	t.machine.Fire(triggerReady)
}

func (t *Task) MutateTo(s TaskState) {
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
	case TaskStateError:
		err = t.machine.Fire(triggerError)
	case TaskStateDegraded:
		err = t.machine.Fire(triggerDegraded)
	case TaskStateExited:
		err = t.machine.Fire(triggerExit)
	case TaskStateInactive:
		err = t.machine.Fire(triggerInactive)
	case TaskStateUnknown:
		err = t.machine.Fire(triggerUnknown)
	}

	if err != nil {
		fmt.Println(err)
		zap.S().Errorw("failed to transition task", "id", t.name, "error", err)
		return
	}

	// mutate the nextState to the current state and let the scheduler decide what to do next
	t.nextState = t.CurrentState()
}

func (t *Task) String() string {
	task := struct {
		Name         string `json:"name"`
		Workload     string `json:"workload"`
		CurrentState string `json:"current_state"`
		NextState    string `json:"next_state"`
	}{
		Name:         t.name,
		Workload:     t.Workload.String(),
		CurrentState: t.CurrentState().String(),
		NextState:    t.NextState().String(),
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
	return t.Hash()
}

func (t *Task) Name() string {
	return t.name
}

func (t *Task) SetMark(key string, val interface{}) {
	t.marks[key] = val
}

func (t *Task) GetMark(key string) (value interface{}, ok bool) {
	value, ok = t.marks[key]
	return
}

func (t *Task) RemoveMark(key string) {
	delete(t.marks, key)
}

func (t *Task) CleanMarks() {
	t.marks = make(map[string]interface{})
}

func (t *Task) GetMarks() []string {
	marks := make([]string, 0, len(t.marks))
	for k := range t.marks {
		marks = append(marks, k)
	}
	return marks
}

func (t *Task) HasMarks() bool {
	return len(t.marks) > 0
}

func (t *Task) initStateless() *stateless.StateMachine {
	machine := stateless.NewStateMachine(TaskStateReady)
	machine.Configure(TaskStateReady).
		Permit(triggerDeploy, TaskStateDeploying).
		Permit(triggerInactive, TaskStateInactive)

	machine.Configure(TaskStateDeploying).
		Permit(triggerDeployed, TaskStateDeployed).
		Permit(triggerExit, TaskStateExited).
		Permit(triggerError, TaskStateError)

	machine.Configure(TaskStateError).
		Permit(triggerReady, TaskStateReady)

	machine.Configure(TaskStateDeployed).
		Permit(triggerRun, TaskStateRunning).
		Permit(triggerExit, TaskStateExited).
		Permit(triggerUnknown, TaskStateUnknown).
		Permit(triggerStop, TaskStateStopping)

	machine.Configure(TaskStateRunning).
		Permit(triggerStop, TaskStateStopping).
		Permit(triggerExit, TaskStateExited).
		Permit(triggerUnknown, TaskStateUnknown)

	machine.Configure(TaskStateStopping).
		Permit(triggerReady, TaskStateReady).
		Permit(triggerExit, TaskStateExited)

	machine.Configure(TaskStateStopped).
		Permit(triggerInactive, TaskStateInactive).
		Permit(triggerReady, TaskStateReady)

	machine.Configure(TaskStateExited).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			t.Failures++
			return nil
		}).
		Permit(triggerInactive, TaskStateInactive).
		Permit(triggerReady, TaskStateReady)

	machine.Configure(TaskStateUnknown).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			t.Failures++
			return nil
		}).
		Permit(triggerDeploy, TaskStateDeploying).
		Permit(triggerInactive, TaskStateInactive).
		Permit(triggerReady, TaskStateReady)

	machine.Configure(TaskStateInactive).
		Permit(triggerReady, TaskStateReady)

	machine.OnTransitioned(func(ctx context.Context, tt stateless.Transition) {
		zap.S().Debugf("task %s transitioned from %s to %s", t.name, tt.Source, tt.Destination)
	})

	return machine
}
