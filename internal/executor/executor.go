package executor

import (
	"context"
	"os"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
	"go.uber.org/zap"
)

type WorkloadInfo struct {
	Id     string
	Name   string
	Status string
}

type Podman interface {
	List() ([]WorkloadInfo, error)
	Remove(workloadId string) error
	Run(manifestPath, authFilePath string, annotations map[string]string) ([]*PodReport, error)
	Start(workloadId string) error
	Stop(workloadId string) error
	Exists(workloadId string) (bool, error)
}

type Executor struct {
	podman  Podman
	futures map[string]chan task.State
	ids     map[string]string
}

func New() (*Executor, error) {
	podman, err := NewPodman(os.Getenv("XDG_RUNTIME_DIR"))
	if err != nil {
		return nil, err
	}
	return &Executor{
		podman:  podman,
		futures: make(map[string]chan task.State),
		ids:     make(map[string]string),
	}, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) *scheduler.Future[task.State] {
	workload := w.(entity.PodWorkload)

	ch := make(chan task.State)
	future := scheduler.NewFuture(ch)
	e.futures[w.ID()] = ch

	pod, err := toPod(workload)
	if err != nil {
		zap.S().Errorw("failed to create pod", "error", err)
		e.sendState(w.ID(), task.ExitedState, true)
		return future
	}

	yaml, err := toPodYaml(pod, workload.Configmaps)
	if err != nil {
		zap.S().Errorw("failed to create pod", "error", err)
		e.sendState(w.ID(), task.ExitedState, true)
		return future
	}

	zap.S().Debugw("pod spec", "spec", string(yaml))

	// save file
	tmp, _ := os.CreateTemp("/home/cosmin/tmp", "flotta-")
	tmp.Write(yaml)
	tmp.Close()

	report, err := e.podman.Run(tmp.Name(), workload.ImageRegistryAuth, workload.Annotations)
	if err != nil {
		zap.S().Errorw("failed to execute workload", "error", err, "report", report)
		e.sendState(w.ID(), task.ExitedState, true)
		return future
	}

	zap.S().Infow("workload started", "hash", w.Hash(), "report", report)
	ch <- task.DeployedState

	err = e.podman.Start(report[0].Id)
	if err != nil {
		e.sendState(w.ID(), task.ExitedState, true)
		return future
	}

	e.sendState(w.ID(), task.RunningState, false)
	e.ids[w.ID()] = report[0].Id

	return future
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) {
	podID := e.ids[w.ID()]
	err := e.podman.Stop(podID)
	if err != nil {
		zap.S().Errorw("failed to stop pod", "error", err, "pod_id", podID, "workload_id", w.ID())
		return
	}

	err = e.podman.Remove(podID)
	if err != nil {
		zap.S().Errorw("failed to remove pod", "error", err, "pod_id", podID, "workload_id", w.ID())
		return
	}

	e.sendState(w.ID(), task.ExitedState, false)

	zap.S().Infow("workload stopped", "workload_id", w.ID())
}

func (e *Executor) sendState(workloadID string, state task.State, removeFuture bool) {
	ch, found := e.futures[workloadID]
	if !found {
		return
	}

	ch <- state
	if removeFuture {
		close(ch)
		delete(e.futures, workloadID)
	}
}
