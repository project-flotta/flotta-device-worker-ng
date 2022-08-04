package configuration

import (
	"github.com/project-flotta/flotta-operator/models"
	"github.com/tupyy/device-worker-ng/internal/entity"
)

func model2PodWorkload(m models.Workload) entity.PodWorkload {
	pod := entity.PodWorkload{
		Name:          m.Name,
		Namespace:     m.Namespace,
		Annotations:   m.Annotations,
		Labels:        m.Labels,
		Specification: m.Specification,
	}

	if m.ImageRegistries != nil {
		pod.ImageRegistryAuth = m.ImageRegistries.AuthFile
	}

	pod.Configmaps = make([]string, 0, len(m.Configmaps))
	for _, c := range m.Configmaps {
		pod.Configmaps = append(pod.Configmaps, c)
	}

	return pod
}
