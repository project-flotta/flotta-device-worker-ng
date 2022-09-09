package executor

import (
	"context"
	"fmt"

	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/executor/k8s"
	"github.com/tupyy/device-worker-ng/internal/executor/podman"
	"go.uber.org/zap"
)

// executor is defines the interface for all executors: podman, bash, ansible.
type executor interface {
	Remove(ctx context.Context, w entity.Workload) error
	Run(ctx context.Context, w entity.Workload) error
	Stop(ctx context.Context, id string) error
	GetState(ctx context.Context, w entity.Workload) (entity.JobState, error)
	Exists(ctx context.Context, id string) (bool, error)
}

type executorItem[T any] struct {
	Name     string
	Rootless bool
	Kind     entity.WorkloadKind
	Value    T
}

type Executor struct {
	executors []executorItem[executor]
	ids       map[string]string
}

func New() (*Executor, error) {
	e := &Executor{
		executors: make([]executorItem[executor], 0),
		ids:       make(map[string]string),
	}
	e.createExecutors()
	return e, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) error {
	executor, err := e.getExecutor(w)
	if err != nil {
		return err
	}

	return executor.Run(ctx, w)
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) error {
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
	executor, err := e.getExecutor(w)
	if err != nil {
		return entity.UnknownState, err
	}
	state, err := executor.GetState(ctx, w)
	if err != nil {
		zap.S().Errorw("failed to get workload status", "error", err)
		return entity.UnknownState, err
	}
	return state, nil
}

func (e *Executor) Remove(ctx context.Context, w entity.Workload) error {
	executor, err := e.getExecutor(w)
	if err != nil {
		return err
	}
	err = executor.Remove(ctx, w)
	if err != nil {
		zap.S().Errorw("failed to get remove workload", "error", err)
		return err
	}
	return nil
}

func (e *Executor) Exists(ctx context.Context, w entity.Workload) (bool, error) {
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
	fn := func(ex []executorItem[executor], kind entity.WorkloadKind, rootless bool) (executor, bool) {
		for _, item := range ex {
			if item.Kind == kind && item.Rootless == rootless {
				return item.Value, true
			}
		}
		return nil, false
	}

	if w.Kind() == entity.PodKind {
		i, found := fn(e.executors, w.Kind(), w.IsRootless())
		if !found {
			return nil, fmt.Errorf("podman executor not found for workload '%s'", w.ID())
		}
		return i, nil
	}

	// for k8s there is no such rootless or rootfull executor. just rootfull
	k8sEx, found := fn(e.executors, w.Kind(), false)
	if !found {
		return nil, fmt.Errorf("k8s executor not found for workload '%s'", w.ID())
	}
	return k8sEx, nil
}

func (e *Executor) createExecutors() {
	// create rootless podman
	if config.GetXDGRuntimeDir() != "" {
		if rootlessPodman, err := podman.New(config.GetXDGRuntimeDir()); err != nil {
			zap.S().Errorw("failed to create podman rootless executor", "error", err)
		} else {
			e.executors = append(e.executors, executorItem[executor]{
				Name:     "podman",
				Rootless: true,
				Kind:     entity.PodKind,
				Value:    rootlessPodman,
			})
			zap.S().Info("podman rootless executor created")
		}
	}

	// create rootfull podman
	if rootfullPodman, err := podman.New("/run"); err != nil {
		zap.S().Errorw("failed to create podman rootfull executor", "error", err)
	} else {
		e.executors = append(e.executors, executorItem[executor]{
			Name:     "podman",
			Rootless: false,
			Kind:     entity.PodKind,
			Value:    rootfullPodman,
		})
		zap.S().Info("podman rootfull executor created")
	}

	// create k8s executor
	if config.GetKubeConfig() != "" {
		if k8sExecutor, err := k8s.New(config.GetKubeConfig()); err != nil {
			zap.S().Errorw("failed to create k8s executor", "error", err)
		} else {
			e.executors = append(e.executors, executorItem[executor]{
				Name:     "k8s",
				Rootless: false,
				Kind:     entity.K8SKind,
				Value:    k8sExecutor,
			})
			zap.S().Info("k8s executor created")
		}
	}
}
