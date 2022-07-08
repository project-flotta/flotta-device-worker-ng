package configuration

import (
	"time"

	"github.com/tupyy/device-worker-ng/internal/entities"
)

type Manager struct {
	conf     entities.DeviceConfiguration
	hardware entities.HardwareInfo
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

	m := &Manager{conf: c}
	m.hardware = m.GetHardwareInfo()

	return m
}

func (c *Manager) Configuration() entities.DeviceConfiguration {
	return c.conf
}

func (c *Manager) SetConfiguration(e entities.DeviceConfiguration) {
	c.conf = e
}

func (c *Manager) GetHardwareInfo() entities.HardwareInfo {
	return c.hardware
}

func (c *Manager) Heartbeat() entities.Heartbeat {
	return entities.Heartbeat{
		Hardware: &c.hardware,
	}
}
