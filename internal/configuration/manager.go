package configuration

import (
	"time"

	"github.com/tupyy/device-worker-ng/internal/entities"
)

type Manager struct {
}

func (c *Manager) Configuration() entities.DeviceConfiguration {
	return entities.DeviceConfiguration{
		Heartbeat: entities.HeartbeatConfiguration{
			HardwareProfile: entities.HardwareProfileConfiguration{
				Include: true,
				Scope:   entities.FullScope,
			},
			Period: 1 * time.Second,
		},
	}
}

func (c *Manager) GetHardwareInfo() entities.HardwareInfo {
	h := NewHardwareInfo(nil)
	return h.GetHardwareInformation()
}

func (c *Manager) Heartbeat() entities.Heartbeat {
	h := c.GetHardwareInfo()
	return entities.Heartbeat{
		Hardware: &h,
	}
}
