package entity

import "time"

type Heartbeat struct {
	// Events produced by device worker.
	Events []*EventInfo

	// hardware
	Hardware *HardwareInfo

	// status
	// Enum: [up degraded]
	Status HearbeatStatus

	// upgrade
	Upgrade *UpgradeStatus

	// version
	Version string

	// workloads
	Workloads []*WorkloadStatus
}

type EventInfo struct {
	// Message describe the event which has occured.
	Message string

	// Reason is single word description of the subject of the event.
	Reason string

	// Either 'info' or 'warn', which reflect the importance of event.
	// Enum: [info warn]
	Type EventType
}

type UpgradeStatus struct {
	// current commit ID
	CurrentCommitID string

	// last upgrade status
	// Enum: [succeeded failed]
	LastUpgradeStatus string

	// last upgrade time
	LastUpgradeTime string
}

type WorkloadStatus struct {
	// last data upload
	LastDataUpload time.Time

	// name
	Name string

	// status
	// Enum: [deploying running crashed stopped]
	Status Status
}

type Status int

func (s Status) String() string {
	switch s {
	case Deploying:
		return "deploying"
	case Running:
		return "running"
	case Crashed:
		return "crashed"
	case Stopped:
		return "stopped"
	default:
		return "unknown"
	}
}

const (
	Deploying Status = iota
	Running
	Crashed
	Stopped
)

type HearbeatStatus int

func (hs HearbeatStatus) String() string {
	switch hs {
	case Up:
		return "up"
	case Degraded:
		return "degraded"
	default:
		return "unknown"
	}
}

const (
	Up HearbeatStatus = iota
	Degraded
)

type EventType int

func (e EventType) String() string {
	switch e {
	case Info:
		return "info"
	case Warn:
		return "warn"
	default:
		return "unknown"
	}
}

const (
	Info EventType = iota
	Warn
)
