package configuration

import (
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
	// SchedulerCh is the channel to communicate with the scheduler
	SchedulerCh chan entity.Message
	// StateManagerCh is the channel to communicate with state manager
	StateManagerCh chan entity.Message

	conf     entity.DeviceConfigurationMessage
	hardware entity.HardwareInfo
	lock     sync.Mutex
}

func New() *Manager {
	m := &Manager{
		conf:           defaultConfiguration,
		SchedulerCh:    make(chan entity.Message, 10),
		StateManagerCh: make(chan entity.Message),
		hardware:       NewHardwareInfo(nil).GetHardwareInformation(),
	}

	return m
}

func (c *Manager) HardwareInfo() entity.HardwareInfo {
	return c.hardware
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

	// send task to scheduler
	o := entity.Option[[]entity.Workload]{
		Value: e.Workloads,
	}

	if len(e.Workloads) == 0 {
		o.None = true
	}

	zap.S().Debugw("new workloads", "workloads", o)
	c.SchedulerCh <- entity.Message{
		Kind:    entity.WorkloadConfigurationMessage,
		Payload: o,
	}

	// send profiles to state manager
	if deviceProfiles, err := c.createDeviceProfiles(c.conf); err != nil {
		zap.S().Errorw("cannot parse profiles", "error", err)
	} else {
		c.StateManagerCh <- entity.Message{Kind: entity.ProfileConfigurationMessage, Payload: deviceProfiles}
	}

	c.conf = e
}

func (c *Manager) Heartbeat() entity.Heartbeat {
	return entity.Heartbeat{
		Hardware: &c.hardware,
	}
}

// create a list of device profiles from DeviceConfigurationMessage
// It returns a list with all profiles or error if one expression is not valid.
func (c *Manager) createDeviceProfiles(conf entity.DeviceConfigurationMessage) (entity.Option[map[string]entity.DeviceProfile], error) {
	return entity.Option[map[string]entity.DeviceProfile]{
		Value: map[string]entity.DeviceProfile{},
		None:  true,
	}, nil
}
