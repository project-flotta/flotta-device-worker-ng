package entities

import "time"

type Heartbeat struct {
}

type HeartbeatConfiguration struct {
	HardwareProfile HardwareProfileConfiguration

	// period in seconds
	Period time.Duration
}

type Scope int

const (
	FullScope Scope = iota
	DeltaScope
)

type HardwareProfileConfiguration struct {
	Include bool
	Scope   Scope
}
