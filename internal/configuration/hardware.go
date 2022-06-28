package configuration

import (
	"github.com/openshift/assisted-installer-agent/src/inventory"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/tupyy/device-worker-ng/internal/entities"
)

type HardwareInfo struct {
	dependencies util.IDependencies
}

func NewHardwareInfo(dep util.IDependencies) *HardwareInfo {
	var d util.IDependencies
	if dep == nil {
		d = util.NewDependencies("/")
	} else {
		d = dep
	}

	return &HardwareInfo{d}
}

func (h *HardwareInfo) GetHardwareInformation() entities.HardwareInfo {
	hardwareInfo := entities.HardwareInfo{}

	h.getHardwareImmutableInformation(&hardwareInfo)
	h.getHardwareMutableInformation(&hardwareInfo)

	return hardwareInfo
}

func (h *HardwareInfo) getHardwareImmutableInformation(hardwareInfo *entities.HardwareInfo) {
	cpu := inventory.GetCPU(h.dependencies)
	systemVendor := inventory.GetVendor(h.dependencies)

	hardwareInfo.CPU = entities.CPU{
		Architecture: cpu.Architecture,
		ModelName:    cpu.ModelName,
		Flags:        []string{},
	}

	hardwareInfo.SystemVendor = entities.SystemVendor{
		Manufacturer: systemVendor.Manufacturer,
		ProductName:  systemVendor.ProductName,
		SerialNumber: systemVendor.SerialNumber,
		Virtual:      systemVendor.Virtual,
	}
}

func (h *HardwareInfo) getHardwareMutableInformation(hardwareInfo *entities.HardwareInfo) error {
	hostname := inventory.GetHostname(h.dependencies)
	interfaces := inventory.GetInterfaces(h.dependencies)

	hardwareInfo.Hostname = hostname
	for _, currInterface := range interfaces {
		if len(currInterface.IPV4Addresses) == 0 && len(currInterface.IPV6Addresses) == 0 {
			continue
		}
		newInterface := entities.Interface{
			IPV4Addresses: currInterface.IPV4Addresses,
			IPV6Addresses: currInterface.IPV6Addresses,
			Flags:         []string{},
		}
		hardwareInfo.Interfaces = append(hardwareInfo.Interfaces, newInterface)
	}

	return nil
}
