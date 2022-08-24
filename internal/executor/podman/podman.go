package podman

import (
	"context"
	"fmt"

	podmanEvents "github.com/containers/podman/v4/libpod/events"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/play"
	"github.com/containers/podman/v4/pkg/bindings/pods"
	"github.com/tupyy/device-worker-ng/internal/executor/common"
	"go.uber.org/zap"
)

const (
	StoppedContainer = "stoppedContainer"
	StartedContainer = "startedContainer"

	DefaultNetworkName = "podman"

	podmanStart  = string(podmanEvents.Start)
	podmanRemove = string(podmanEvents.Remove)
	podmanStop   = string(podmanEvents.Stop)

	podmanBinary = "/usr/bin/podman"
)

const (
	DefaultTimeoutForStoppingInSeconds int = 5
)

var (
	boolFalse = false
)

type ContainerReport struct {
	IPAddress string
	Id        string
	Name      string
}

type PodReport struct {
	Id         string
	Name       string
	Containers []*ContainerReport
}

func (p *PodReport) AppendContainer(c *ContainerReport) {
	p.Containers = append(p.Containers, c)
}

type podman struct {
	podmanConnection   context.Context
	timeoutForStopping int
}

func NewPodman(xdgRuntimeDir string) (*podman, error) {
	podmanConnection, err := podmanConnection(xdgRuntimeDir)
	if err != nil {
		return nil, err
	}
	p := &podman{
		podmanConnection:   podmanConnection,
		timeoutForStopping: DefaultTimeoutForStoppingInSeconds,
	}

	return p, nil
}

func (p *podman) List() ([]common.WorkloadInfo, error) {
	podList, err := pods.List(p.podmanConnection, nil)
	if err != nil {
		return nil, err
	}
	var workloads []common.WorkloadInfo
	for _, pod := range podList {
		wi := common.WorkloadInfo{
			Id:     pod.Name,
			Name:   pod.Name,
			Status: pod.Status,
		}
		workloads = append(workloads, wi)
	}
	return workloads, nil
}

func (p *podman) Exists(workloadId string) (bool, error) {
	exists, err := pods.Exists(p.podmanConnection, workloadId, nil)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (p *podman) Remove(workloadId string) error {
	exists, err := p.Exists(workloadId)
	if err != nil {
		return err
	}
	if exists {
		force := true
		_, err := pods.Remove(p.podmanConnection, workloadId, &pods.RemoveOptions{Force: &force})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *podman) Run(manifestPath, authFilePath string, annotations map[string]string) ([]*PodReport, error) {
	network := []string{DefaultNetworkName}
	options := play.KubeOptions{
		Authfile:    &authFilePath,
		Network:     &network,
		Annotations: annotations,
		Start:       &boolFalse,
	}
	report, err := play.Kube(p.podmanConnection, manifestPath, &options)
	if err != nil {
		return nil, err
	}

	var podIds = make([]*PodReport, len(report.Pods))
	for i, pod := range report.Pods {
		report := &PodReport{Id: pod.ID}
		for _, container := range pod.Containers {
			c, err := p.getContainerDetails(container)
			if err != nil {
				zap.S().Errorf("cannot get container information: %v", err)
				continue
			}
			report.AppendContainer(c)
		}
		podIds[i] = report
	}
	return podIds, nil
}

func (p *podman) Start(workloadId string) error {
	_, err := pods.Start(p.podmanConnection, workloadId, nil)
	if err != nil {
		return err
	}
	return nil
}

func (p *podman) Stop(workloadId string) error {
	exists, err := pods.Exists(p.podmanConnection, workloadId, nil)
	if err != nil {
		return err
	}
	if exists {
		_, err := pods.Stop(p.podmanConnection, workloadId, &pods.StopOptions{Timeout: &p.timeoutForStopping})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *podman) getContainerDetails(containerId string) (*ContainerReport, error) {
	data, err := containers.Inspect(p.podmanConnection, containerId, nil)
	if err != nil {
		return nil, err
	}
	// this is the default one
	network, ok := data.NetworkSettings.Networks[DefaultNetworkName]
	if !ok {
		return nil, fmt.Errorf("cannot retrieve container '%s' network information", containerId)
	}

	return &ContainerReport{
		IPAddress: network.InspectBasicNetworkConfig.IPAddress,
		Id:        containerId,
		Name:      data.Name,
	}, nil
}

func podmanConnection(xdgRuntimeDir string) (context.Context, error) {
	podmanConnection, err := bindings.NewConnection(context.Background(), fmt.Sprintf("unix:%s/podman/podman.sock", xdgRuntimeDir))
	if err != nil {
		return nil, err
	}

	return podmanConnection, err
}
