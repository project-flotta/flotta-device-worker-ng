package configuration

import (
	"errors"
	"sync"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
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
	// TaskCh is the channel where task are sent
	TaskCh chan map[string]entity.Task
	// ProfileCh is the channel where device profiles are sent
	ProfileCh chan map[string]entity.DeviceProfile

	conf     entity.DeviceConfigurationMessage
	hardware entity.HardwareInfo
	lock     sync.Mutex
}

func New() *Manager {
	m := &Manager{
		conf:      defaultConfiguration,
		TaskCh:    make(chan map[string]entity.Task),
		ProfileCh: make(chan map[string]entity.DeviceProfile),
	}

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

	c.TaskCh <- c.createTasks(c.conf)

	if deviceProfiles, err := c.createDeviceProfiles(c.conf); err != nil {
		zap.S().Errorw("cannot parse profiles", "error", err)
	} else {
		c.ProfileCh <- deviceProfiles
	}

	c.conf = e
}

func (c *Manager) Heartbeat() entity.Heartbeat {
	return entity.Heartbeat{
		Hardware: &c.hardware,
	}
}

// createTasks creates a list of task from workload definition
func (c *Manager) createTasks(conf entity.DeviceConfigurationMessage) map[string]entity.Task {
	return map[string]entity.Task{}
}

// create a list of device profiles from DeviceConfigurationMessage
// It returns a list with all profiles or error if one expression is not valid.
func (c *Manager) createDeviceProfiles(conf entity.DeviceConfigurationMessage) (map[string]entity.DeviceProfile, error) {
	return map[string]entity.DeviceProfile{}, errors.New("not implemented yet")
}
