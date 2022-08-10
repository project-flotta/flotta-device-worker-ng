package executor

import (
	"context"

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
	podman Podman
}

func New() (*Executor, error) {
	podman, err := NewPodman()
	if err != nil {
		return nil, err
	}
	return &Executor{podman}, nil
}

func (e *Executor) Run(ctx context.Context, w entity.Workload) *scheduler.Future[scheduler.ExecutionResult] {
	zap.S().Infow("executor run called", "workload", w)
	return nil
}

func (e *Executor) Stop(ctx context.Context, w entity.Workload) *scheduler.Future[scheduler.ExecutionResult] {
	zap.S().Infow("executor stop called", "workload", w)
	return nil
}
