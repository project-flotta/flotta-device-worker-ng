package configuration

import (
	"sync"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
)

var (
	// default configuration
	defaultConfiguration = entity.DeviceConfigurationMessage{
		Configuration: entity.DeviceConfiguration{
			Heartbeat: entity.HeartbeatConfiguration{
				HardwareProfile: entity.HardwareProfileConfiguration{
					Include: true,
					Scope:   entity.FullScope,
				},
				Period: 1 * time.Second,
			},
		},
	}
)

type Manager struct {
	conf     entity.DeviceConfigurationMessage
	hardware entity.HardwareInfo
	lock     sync.Mutex
}

func New() *Manager {
	m := &Manager{conf: defaultConfiguration}

	return m
}

func (c *Manager) Configuration() entity.DeviceConfigurationMessage {
	return c.conf
}

func (c *Manager) SetConfiguration(e entity.DeviceConfigurationMessage) {
	if e.Hash() == c.conf.Hash() {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	tasks := c.createTasks(c.conf)

	c.conf = e
}

func (c *Manager) Heartbeat() entity.Heartbeat {
	return entity.Heartbeat{
		Hardware: &c.hardware,
	}
}

// createTasks creates a list of task from workload definition
func (c *Manager) createTasks(conf entity.DeviceConfigurationMessage) map[string]entity.Task {
	for _,, w := range conf.Workloads {

	}
	return map[string]entity.Task{}
}
