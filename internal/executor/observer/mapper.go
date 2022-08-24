package observer

import "github.com/tupyy/device-worker-ng/internal/scheduler/task"

func mapToState(state string) task.State {
	switch state {
	case "Running":
		return task.RunningState
	case "Degraded":
		return task.DegradedState
	case "Stopped":
		return task.StoppedState
	case "Error":
		return task.ErrorState
	case "Exited":
		return task.ExitedState
	case "Created":
		return task.DeployedState
	default:
		return task.UnknownState
	}
}
