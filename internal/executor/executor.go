package executor

import (
	"context"
	"errors"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/executor/common"
	"github.com/tupyy/device-worker-ng/internal/executor/observer"
	"github.com/tupyy/device-worker-ng/internal/executor/podman"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
	"go.uber.org/zap"
)

// executor is defines the interface for all executors: podman, bash, ansible.
type executor interface {
	Remove(ctx context.Context, id string) error
	Run(ctx context.Context, w entity.Workload) (string, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	List(ctx context.Context) ([]common.WorkloadInfo, error)
	Exists(ctx context.Context, id string) (bool, error)
}

type Executor struct {
	executors map[entity.WorkloadKind]executor
	observer  *observer.Observer
	ids       map[string]string
}

func New() (*Executor, error) {
	e := &Executor{
		executors: make(map[entity.WorkloadKind]executor),
		ids:       make(map[string]string),
		observer:  observer.New(),
	}
	podman, err := podman.New(true)
	if err != nil {
		return nil, err
	}
	e.executors[entity.PodKind] = podman
	return e, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) (*scheduler.Future[task.State], error) {
	if w.Kind() != entity.PodKind {
		return nil, errors.New("only pod workloads are supported")
	}
	executor := e.executors[w.Kind()]

	exists, err := executor.Exists(ctx, w.ID())
	if err != nil {
		return nil, err
	}

	if exists {
		ch := e.observer.RegisterWorkload(w.ID(), executor)
		future := scheduler.NewFuture(ch)
		ch <- task.DeployedState
		return future, nil
	}

	_, err = executor.Run(ctx, w)
	if err != nil {
		return nil, err
	}
	zap.S().Infow("workload started", "id", w.ID())

	ch := e.observer.RegisterWorkload(w.ID(), executor)
	future := scheduler.NewFuture(ch)
	ch <- task.DeployedState

	return future, nil
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) {
	if w.Kind() != entity.PodKind {
		zap.S().Errorw("workload type unsupported %s", w.Kind())
		return
	}

	executor := e.executors[w.Kind()]
	if err := executor.Stop(ctx, w.ID()); err != nil {
		zap.S().Errorw("failed to stop workload", "error", err)
		return
	}

	zap.S().Infow("workload stopped", "workload_id", w.ID())
}
