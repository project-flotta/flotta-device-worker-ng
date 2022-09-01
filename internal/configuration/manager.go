package configuration

import (
	"sync"
	"time"

	"github.com/tupyy/device-worker-ng/internal/configuration/interpreter"
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

func (c *Manager) SetConfiguration(newConf entity.DeviceConfigurationMessage) {
	if newConf.Hash() == c.conf.Hash() {
		return
	}

	zap.S().Debugw("configurations", "old conf", c.conf, "new conf", newConf)

	// send task to scheduler
	o := entity.Option[[]entity.Workload]{
		Value: newConf.Workloads,
	}

	if len(newConf.Workloads) == 0 {
		o.None = true
	}

	c.SchedulerCh <- entity.Message{
		Kind:    entity.WorkloadConfigurationMessage,
		Payload: o,
	}

	// send profiles to state manager
	if deviceProfiles, err := c.createDeviceProfiles(newConf.Configuration.Profiles); err != nil {
		zap.S().Errorw("cannot parse profiles", "error", err)
	} else {
		c.StateManagerCh <- entity.Message{Kind: entity.ProfileConfigurationMessage, Payload: deviceProfiles}
	}

	c.conf = newConf
}

func (c *Manager) Heartbeat() entity.Heartbeat {
	return entity.Heartbeat{
		Hardware: &c.hardware,
	}
}

// create a list of device profiles from DeviceConfigurationMessage
// It returns a list with all profiles or error if one expression is not valid.
func (c *Manager) createDeviceProfiles(confProfiles map[string]map[string]string) (entity.Option[map[string]entity.DeviceProfile], error) {
	if len(confProfiles) == 0 {
		return entity.Option[map[string]entity.DeviceProfile]{
			Value: map[string]entity.DeviceProfile{},
			None:  true,
		}, nil
	}

	profiles := make(map[string]entity.DeviceProfile)
	for name, conditions := range confProfiles {
		d := entity.DeviceProfile{
			Name:       name,
			Conditions: make([]entity.ProfileCondition, 0, len(conditions)),
		}
		for name, expression := range conditions {
			intr, err := interpreter.New(expression)
			if err != nil {
				zap.S().Errorw("failed to interpret expression", "expression", expression, "error", err)
				break
			}
			d.Conditions = append(d.Conditions, entity.ProfileCondition{
				Name:       name,
				Expression: intr,
			})
		}
		profiles[name] = d
	}

	return entity.Option[map[string]entity.DeviceProfile]{
		Value: profiles,
		None:  len(profiles) == 0,
	}, nil

}
