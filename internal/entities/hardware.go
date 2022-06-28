package entities

type HardwareInfo struct {
	Boot         Boot
	CPU          CPU
	Disks        []Disk
	Gpus         []Gpu
	HostDevices  []HostDevice
	Hostname     string
	Interfaces   []Interface
	Memory       Memory
	Mounts       []Mount
	SystemVendor SystemVendor
}

type Boot struct {
	// current boot mode
	CurrentBootMode string

	// pxe interface
	PxeInterface string
}

type CPU struct {
	// architecture
	Architecture string

	// count
	Count int64

	// flags
	Flags []string

	// frequency
	Frequency float64

	// model name
	ModelName string
}

type Disk struct {
	// bootable
	Bootable bool

	// by-id is the World Wide Number of the device which guaranteed to be unique for every storage device
	ByID string

	// by-path is the shortest physical path to the device
	ByPath string

	// drive type
	DriveType string

	// hctl
	Hctl string

	// Determine the disk's unique identifier which is the by-id field if it exists and fallback to the by-path field otherwise
	ID string

	// io perf
	IoPerf *IoPerf

	// Whether the disk appears to be an installation media or not
	IsInstallationMedia bool

	// model
	Model string

	// name
	Name string

	// path
	Path string

	// serial
	Serial string

	// size bytes
	SizeBytes int64

	// smart
	Smart string

	// vendor
	Vendor string

	// wwn
	Wwn string
}

type IoPerf struct {
	// 99th percentile of fsync duration in milliseconds
	SyncDuration int64
}

type Gpu struct {

	// Device address (for example "0000:00:02.0")
	Address string

	// ID of the device (for example "3ea0")
	DeviceID string

	// Product name of the device (for example "UHD Graphics 620 (Whiskey Lake)")
	Name string

	// The name of the device vendor (for example "Intel Corporation")
	Vendor string

	// ID of the vendor (for example "8086")
	VendorID string
}

type HostDevice struct {

	// Type of the device
	DeviceType string

	// Group id
	Gid int64

	// Major of the device
	Major int64

	// Minor of the device
	Minor int64

	// Path of the device
	Path string

	// Owner id
	UID int64
}

type Interface struct {

	// biosdevname
	Biosdevname string

	// client id
	ClientID string

	// flags
	Flags []string

	// has carrier
	HasCarrier bool

	// ipv4 addresses
	IPV4Addresses []string

	// ipv6 addresses
	IPV6Addresses []string

	// mac address
	MacAddress string

	// mtu
	Mtu int64

	// name
	Name string

	// product
	Product string

	// speed mbps
	SpeedMbps int64

	// vendor
	Vendor string
}

type Memory struct {

	// physical bytes
	PhysicalBytes int64

	// usable bytes
	UsableBytes int64
}

type SystemVendor struct {

	// manufacturer
	Manufacturer string

	// product name
	ProductName string

	// serial number
	SerialNumber string

	// Whether the machine appears to be a virtual machine or not
	Virtual bool
}
