package executor

import (
	"context"
	"os"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
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
	futures map[string]chan scheduler.TaskState
}

func New() (*Executor, error) {
	podman, err := NewPodman(os.Getenv("XDG_RUNTIME_DIR"))
	if err != nil {
		return nil, err
	}
	return &Executor{
		podman:  podman,
		futures: make(map[string]chan scheduler.TaskState),
	}, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) *scheduler.Future[scheduler.TaskState] {
	zap.S().Infow("executor run called", "workload", w)
	pod := w.(entity.PodWorkload)

	ch := make(chan scheduler.TaskState)
	future := scheduler.NewFuture(ch)
	e.futures[w.Hash()] = ch
	report, err := e.podman.Run(pod.Specification, pod.ImageRegistryAuth, pod.Annotations)
	if err != nil {
		zap.S().Errorw("failed to execute workload", "error", err, "report", report)
		ch <- scheduler.TaskStateExited
		return future
	}

	zap.S().Infow("workload started", "hash", w.Hash(), "report", report)
	ch <- scheduler.TaskStateDeployed

	return future
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) {
	zap.S().Infow("executor stop called", "workload", w)
}
