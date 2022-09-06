package executor

import (
	"context"
	"errors"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/executor/podman"
	"go.uber.org/zap"
)

const (
	rootless = true
	rootfull = false
)

// executor is defines the interface for all executors: podman, bash, ansible.
type executor interface {
	Remove(ctx context.Context, id string) error
	Run(ctx context.Context, w entity.Workload) error
	Stop(ctx context.Context, id string) error
	GetState(ctx context.Context, id string) (entity.JobState, error)
	Exists(ctx context.Context, id string) (bool, error)
}

type Executor struct {
	rootlessExecutors map[entity.WorkloadKind]executor
	rootfullExecutors map[entity.WorkloadKind]executor
	ids               map[string]string
}

func New() (*Executor, error) {
	e := &Executor{
		rootlessExecutors: make(map[entity.WorkloadKind]executor),
		rootfullExecutors: make(map[entity.WorkloadKind]executor),
		ids:               make(map[string]string),
	}
	rootlessPodman, err := podman.New(rootless)
	if err != nil {
		return nil, err
	}
	e.rootlessExecutors[entity.PodKind] = rootlessPodman
	rootfullPodman, err := podman.New(rootfull)
	if err != nil {
		return nil, err
	}
	e.rootfullExecutors[entity.PodKind] = rootfullPodman
	return e, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) error {
	if w.Kind() != entity.PodKind {
		return errors.New("only pod workloads are supported")
	}
	executor, err := e.getExecutor(w)
	if err != nil {
		return err
	}

	return executor.Run(ctx, w)
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) error {
	if w.Kind() != entity.PodKind {
		zap.S().Errorw("workload type unsupported %s", w.Kind())
		return errors.New("only pod workloads are supported")
	}

	executor, err := e.getExecutor(w)
	if err != nil {
		return err
	}
	if err := executor.Stop(ctx, w.ID()); err != nil {
		zap.S().Errorw("failed to stop workload", "error", err)
		return err
	}

	zap.S().Infow("workload stopped", "workload_id", w.ID())

	return nil
}

func (e *Executor) GetState(ctx context.Context, w entity.Workload) (entity.JobState, error) {
	if w.Kind() != entity.PodKind {
		zap.S().Errorw("workload type unsupported %s", w.Kind())
		return entity.UnknownState, errors.New("only pod workloads are supported")
	}

	executor, err := e.getExecutor(w)
	if err != nil {
		return entity.UnknownState, err
	}
	state, err := executor.GetState(ctx, w.ID())
	if err != nil {
		zap.S().Errorw("failed to get workload status", "error", err)
		return entity.UnknownState, err
	}
	return state, nil
}

func (e *Executor) Remove(ctx context.Context, w entity.Workload) error {
	if w.Kind() != entity.PodKind {
		zap.S().Errorw("workload type unsupported %s", w.Kind())
		return errors.New("only pod workloads are supported")
	}

	executor, err := e.getExecutor(w)
	if err != nil {
		return err
	}
	err = executor.Remove(ctx, w.ID())
	if err != nil {
		zap.S().Errorw("failed to get remove workload", "error", err)
		return err
	}
	return nil
}

func (e *Executor) Exists(ctx context.Context, w entity.Workload) (bool, error) {
	if w.Kind() != entity.PodKind {
		zap.S().Errorw("workload type unsupported %s", w.Kind())
		return false, errors.New("only pod workloads are supported")
	}

	executor, err := e.getExecutor(w)
	if err != nil {
		return false, err
	}
	exists, err := executor.Exists(ctx, w.ID())
	if err != nil {
		zap.S().Errorw("failed to get remove workload", "error", err)
		return false, err
	}
	return exists, nil
}

func (e *Executor) getExecutor(w entity.Workload) (executor, error) {
	if w.IsRootless() {
		ex, found := e.rootlessExecutors[w.Kind()]
		if !found {
			return nil, errors.New("rootless executor not found for this kind of job")
		}
		return ex, nil
	}
	ex, found := e.rootfullExecutors[w.Kind()]
	if !found {
		return nil, errors.New("rootfull executor not found for this kind of job")
	}
	return ex, nil
}
