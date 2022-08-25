package podman

import (
	"context"
	"fmt"
	"os"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/executor/common"
	"go.uber.org/zap"
)

type PodmanExecutor struct {
	podman *podman
	ids    map[string]string
}

func New(rootless bool) (*PodmanExecutor, error) {
	podman, err := NewPodman(os.Getenv("XDG_RUNTIME_DIR"))
	if err != nil {
		return nil, err
	}
	return &PodmanExecutor{
		podman: podman,
		ids:    make(map[string]string),
	}, nil
}

func (e *PodmanExecutor) Run(ctx context.Context, w entity.Workload) error {
	workload := w.(entity.PodWorkload)

	pod, err := toPod(workload)
	if err != nil {
		zap.S().Errorw("failed to create pod", "error", err)
		return fmt.Errorf("[%w] [%s] workload_name '%s'", common.ErrDeployingWorkload, err, workload.Name)
	}

	yaml, err := toPodYaml(pod, workload.Configmaps)
	if err != nil {
		zap.S().Errorw("failed to create pod", "error", err)
		return fmt.Errorf("[%w] [%s] workload_name '%s'", common.ErrDeployingWorkload, err, workload.Name)
	}

	zap.S().Debugw("pod spec", "spec", string(yaml))

	// save file
	tmp, _ := os.CreateTemp("/home/cosmin/tmp", "flotta-")
	tmp.Write(yaml)
	tmp.Close()

	report, err := e.podman.Run(tmp.Name(), workload.ImageRegistryAuth, workload.Annotations)
	if err != nil {
		zap.S().Errorw("failed to execute workload", "error", err, "report", report)
		return fmt.Errorf("%w %s workload_name '%s'", common.ErrDeployingWorkload, err, workload.Name)
	}

	zap.S().Infow("workload started", "hash", w.Hash(), "report", report)

	err = e.podman.Start(report[0].Id)
	if err != nil {
		return fmt.Errorf("%w workload name '%s', error %s", common.ErrRunningWorkload, workload.Name, err)
	}

	return nil
}

func (e *PodmanExecutor) Exists(ctx context.Context, id string) (bool, error) {
	return e.podman.Exists(id)
}

func (e *PodmanExecutor) Start(ctx context.Context, id string) error {
	err := e.podman.Start(id)
	if err != nil {
		return fmt.Errorf("%w workload id '%s', error %s", common.ErrRunningWorkload, id, err)
	}
	zap.S().Infow("workload started", "workload_id", id)
	return nil
}

func (e *PodmanExecutor) Stop(ctx context.Context, id string) error {
	err := e.podman.Stop(id)
	if err != nil {
		zap.S().Errorw("failed to stop pod", "error", err, "pod_id", id)
		return fmt.Errorf("%w %s pod_id: %s", common.ErrStoppingWorkload, err, id)
	}
	zap.S().Infow("workload stopped", "workload_id", id)
	return nil
}

func (e *PodmanExecutor) Remove(ctx context.Context, id string) error {
	err := e.podman.Remove(id)
	if err != nil {
		zap.S().Errorw("failed to remove pod", "error", err, "pod_id", id)
		return fmt.Errorf("%w %s pod_id: %s", common.ErrRemoveWorkload, err, id)
	}
	zap.S().Infow("workload removed", "workload_id", id)
	return nil
}

func (e *PodmanExecutor) List(ctx context.Context) ([]common.WorkloadInfo, error) {
	reports, err := e.podman.List()
	if err != nil {
		return []common.WorkloadInfo{}, err
	}
	return reports, nil
}

func (e *PodmanExecutor) GetState(ctx context.Context, id string) (string, error) {
	info, err := e.List(ctx)
	if err != nil {
		return "", err
	}
	for _, i := range info {
		if i.Id == id {
			return i.Status, nil
		}
	}
	return "unknown", nil
}
