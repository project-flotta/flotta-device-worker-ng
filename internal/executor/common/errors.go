package common

import "errors"

var (
	ErrDeployingWorkload = errors.New("failed to deploy workload")
	ErrRunningWorkload   = errors.New("failed to execute workload")
	ErrStoppingWorkload  = errors.New("failed to stop workload")
	ErrRemoveWorkload    = errors.New("failed to remove workload")
)
