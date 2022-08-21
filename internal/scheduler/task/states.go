package task

type State int

func (ts State) String() string {
	switch ts {
	case ReadyState:
		return "ready"
	case DeployingState:
		return "deploying"
	case DeployedState:
		return "deployed"
	case RunningState:
		return "running"
	case StoppingState:
		return "stopping"
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
	case DeletionState:
		return "deletion"
	default:
		return "unknown"
	}
}

func FromString(val string) State {
	switch val {
	case "ready":
		return ReadyState
	case "deploying":
		return DeployingState
	case "deployed":
		return DeployedState
	case "running":
		return RunningState
	case "stopping":
		return StoppingState
	case "stopped":
		return StoppedState
	case "exited":
		return ExitedState
	case "error":
		return ErrorState
	case "inactive":
		return InactiveState
	case "deletion":
		return DeletionState
	default:
		return UnknownState
	}
}

func (ts State) OneOf(states ...State) bool {
	for _, s := range states {
		if ts == s {
			return true
		}
	}
	return false
}

const (
	// ReadyState indicates that the task ready to be deloyed
	ReadyState State = iota
	// DeployingState indicates that the task is currently deploying
	DeployingState
	// DeployedState indicates that the task has been deployed but not started yet.
	DeployedState
	// RunningState indicates that the task is running
	RunningState
	// StoppingState indicates that the task is about to be stopped
	StoppingState
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
	// DeletionState indicates that the task is being removed from the scheduler.
	DeletionState

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
