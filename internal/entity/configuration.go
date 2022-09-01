package entity

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
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
	json, err := json.Marshal(m)
	if err != nil {
		return err.Error()
	}
	return string(json)
}

func (m DeviceConfigurationMessage) Hash() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s", m.DeviceID)
	fmt.Fprintf(&sb, "%s", m.Version)
	fmt.Fprintf(&sb, "%s", m.WorkloadsMonitoringInterval)
	fmt.Fprintf(&sb, "%s", m.Configuration.String())

	// sort workloads by ID to be sure we don't have surprises
	sort.Slice(m.Workloads, func(i, j int) bool {
		return m.Workloads[i].ID() < m.Workloads[j].ID()
	})
	for _, t := range m.Workloads {
		fmt.Fprintf(&sb, "%s", t.String())
	}
	sum := sha256.Sum256(bytes.NewBufferString(sb.String()).Bytes())
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
	json, err := json.Marshal(d)
	if err != nil {
		return err.Error()
	}
	return string(json)
}

func (d DeviceConfiguration) Hash() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s", d.Heartbeat.String())
	fmt.Fprintf(&sb, "%+v", d.OsInformation)
	for _, m := range d.Mounts {
		fmt.Fprintf(&sb, "%s%s%s%s", m.Device, m.Directory, m.Options, m.Type)
	}
	fmt.Fprintf(&sb, "%+v", d.Profiles)
	sum := sha256.Sum256(bytes.NewBufferString(sb.String()).Bytes())
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
