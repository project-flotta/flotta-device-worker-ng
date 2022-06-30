package entities

import "time"

type HeartbeatConfiguration struct {
	HardwareProfile HardwareProfileConfiguration

	// period in seconds
	Period time.Duration
}

type Scope int

func (s Scope) String() string {
	switch s {
	case FullScope:
		return "full_scope"
	case DeltaScope:
		return "delta"
	default:
		return "unknown"
	}
}

const (
	FullScope Scope = iota
	DeltaScope
)

type HardwareProfileConfiguration struct {
	Include bool
	Scope   Scope
}
