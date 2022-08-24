package common

import "errors"

var (
	ErrorDeployingWorkload = errors.New("failed to deploy workload")
	ErrorRunningWorkload   = errors.New("failed to execute workload")
	ErrorStoppingWorkload  = errors.New("failed to stop workload")
	ErrorRemoveWorkload    = errors.New("failed to remove workload")
)
