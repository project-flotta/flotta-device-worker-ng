package entity

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

type DeviceConfigurationMessage struct {
	// configuration
	Configuration DeviceConfiguration

	// Device identifier
	DeviceID string

	// Version
	Version string

	// list of workloads
	Workloads []Workload

	// Defines the interval in seconds between the attempts to evaluate the workloads status and restart those that failed
	// Minimum: > 0
	WorkloadsMonitoringInterval time.Duration
}

func (m DeviceConfigurationMessage) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "device_id: %s, ", m.DeviceID)
	fmt.Fprintf(&sb, "version: %s, ", m.Version)
	fmt.Fprintf(&sb, "workload monitoring interval: %s, ", m.WorkloadsMonitoringInterval)
	fmt.Fprintf(&sb, "%s, ", m.Configuration.String())
	for _, t := range m.Workloads {
		fmt.Fprintf(&sb, "workload: %s, ", t.String())
	}

	return sb.String()
}

func (m DeviceConfigurationMessage) Hash() string {
	sum := sha256.Sum256(bytes.NewBufferString(m.String()).Bytes())
	return fmt.Sprintf("%x", sum)
}

type DeviceConfiguration struct {
	// Heartbeat configuration
	Heartbeat HeartbeatConfiguration

	// List of user defined mounts
	Mounts []Mount

	// Os information
	OsInformation OsInformation

	Profiles map[string]map[string]string
}

func (d DeviceConfiguration) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "heartbeat: %s, ", d.Heartbeat.String())
	fmt.Fprintf(&sb, "os information: %+v, ", d.OsInformation)
	fmt.Fprintf(&sb, "mounts: , ")
	for _, m := range d.Mounts {
		fmt.Fprintf(&sb, "device: %s\\s", m.Device)
		fmt.Fprintf(&sb, "directory: %s\\s", m.Directory)
		fmt.Fprintf(&sb, "options: %s\\s", m.Options)
		fmt.Fprintf(&sb, "type: %s, ", m.Type)
	}

	return sb.String()
}

func (d DeviceConfiguration) Hash() string {
	sum := sha256.Sum256(bytes.NewBufferString(d.String()).Bytes())
	return fmt.Sprintf("%x", sum)
}

type OsInformation struct {
	// automatically upgrade the OS image
	AutomaticallyUpgrade bool

	// the last commit ID
	CommitID string

	// the URL of the hosted commits web server
	HostedObjectsURL string
}

type Mount struct {
	// path of the device to be mounted
	Device string

	// destination directory
	Directory string

	// mount options
	Options string

	// type of the mount
	Type string
}
