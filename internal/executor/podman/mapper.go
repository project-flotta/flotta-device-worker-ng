package podman

import "github.com/tupyy/device-worker-ng/internal/entity"

func mapPodmanStatusToEntity(state string) entity.JobState {
	switch state {
	case "Running":
		return entity.RunningState
	case "Degraded":
		return entity.DegradedState
	case "Stopped":
		return entity.StoppedState
	case "Error":
		return entity.ErrorState
	case "Exited":
		return entity.ExitedState
	default:
		return entity.UnknownState
	}
}
