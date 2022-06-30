package configuration

import (
	"time"

	"github.com/tupyy/device-worker-ng/internal/entities"
)

type Manager struct {
	conf entities.DeviceConfiguration
}

func New() *Manager {
	c := entities.DeviceConfiguration{
		Heartbeat: entities.HeartbeatConfiguration{
			HardwareProfile: entities.HardwareProfileConfiguration{
				Include: true,
				Scope:   entities.FullScope,
			},
			Period: 1 * time.Second,
		},
	}

	return &Manager{c}
}

func (c *Manager) Configuration() entities.DeviceConfiguration {
	return c.conf
}

func (c *Manager) SetConfiguration(e entities.DeviceConfiguration) {
	c.conf = e
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
