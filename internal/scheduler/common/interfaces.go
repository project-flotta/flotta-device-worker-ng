package common

import (
	"context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
)

type Job interface {
	Name() string
	TargetState() job.State
	CurrentState() job.State
	SetTargetState(state job.State) error
	SetCurrentState(state job.State)
	String() string
	ID() string
	Workload() entity.Workload
	MarkForDeletion()
	IsMarkedForDeletion() bool
}

type Reconciler interface {
	Reconcile(ctx context.Context, jobs []Job, executor Executor)
}

//go:generate mockgen -package=scheduler -destination=mock_executor.go --build_flags=--mod=mod . Executor
type Executor interface {
	Remove(ctx context.Context, w entity.Workload) error
	Run(ctx context.Context, w entity.Workload) error
	Stop(ctx context.Context, w entity.Workload) error
	GetState(ctx context.Context, w entity.Workload) (string, error)
	Exists(ctx context.Context, w entity.Workload) (bool, error)
}
