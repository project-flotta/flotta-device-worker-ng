package entity

import (
	"fmt"
	"strings"
	"time"
)

type HeartbeatConfiguration struct {
	HardwareProfile HardwareProfileConfiguration

	// period in seconds
	Period time.Duration
}

func (h HeartbeatConfiguration) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "hardware profile: %v\n", h.HardwareProfile)
	fmt.Fprintf(&sb, "period: %s", h.Period.String())

	return sb.String()
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
