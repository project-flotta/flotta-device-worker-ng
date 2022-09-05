package entity

type JobState int

const (
	// ReadyState indicates that the task ready to be deloyed
	ReadyState JobState = iota
	// RunningState indicates that the task is running
	RunningState
	// StoppedState indicates that the task has been stopped without error
	StoppedState
	// DegradedState indicates that the task is an degrated state like a pod with containers stopped.
	DegradedState
	// ExitedState indicates that the task has been stopped with an error
	ExitedState
	// ErrorState indicates that deploying of the task has resulted in error.
	ErrorState
	// UnknownState indicates that the task is in an unknown state
	UnknownState
	// InactiveState indicates that the task is in an inactive state.
	InactiveState
)

func (ts JobState) String() string {
	switch ts {
	case ReadyState:
		return "ready"
	case RunningState:
		return "running"
	case DegradedState:
		return "degraded"
	case StoppedState:
		return "stopped"
	case ExitedState:
		return "exited"
	case ErrorState:
		return "error"
	case InactiveState:
		return "inactive"
	default:
		return "unknown"
	}
}

func (ts JobState) OneOf(states ...JobState) bool {
	for _, s := range states {
		if ts == s {
			return true
		}
	}
	return false
}
