package entities

type DeviceConfiguration struct {
	// name of the device master cgroup
	CGroup string

	// Heartbeat configuration
	Heartbeat HeartbeatConfiguration

	// List of user defined mounts
	Mounts []Mount

	// Os information
	OsInformation OsInformation
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
