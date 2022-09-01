package client

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/project-flotta/flotta-operator/models"
	"github.com/tupyy/device-worker-ng/internal/entity"
)

func enrolInfoEntity2Model(e entity.EnrolementInfo) models.EnrolmentInfo {
	m := models.EnrolmentInfo{}

	m.TargetNamespace = &e.TargetNamespace

	hardware := hardwareEntity2Model(e.Features.Hardware)
	m.Features = &models.EnrolmentInfoFeatures{
		Hardware: &hardware,
	}

	return m
}

func registerInfoEntity2Model(e entity.RegistrationInfo) models.RegistrationInfo {
	h := hardwareEntity2Model(e.Hardware)
	return models.RegistrationInfo{
		CertificateRequest: e.CertificateRequest,
		Hardware:           &h,
	}
}

func hardwareEntity2Model(e entity.HardwareInfo) models.HardwareInfo {
	m := models.HardwareInfo{
		Boot: &models.Boot{
			CurrentBootMode: e.Boot.CurrentBootMode,
			PxeInterface:    e.Boot.PxeInterface,
		},
		CPU: &models.CPU{
			Architecture: e.CPU.Architecture,
			Count:        e.CPU.Count,
			Frequency:    e.CPU.Frequency,
			ModelName:    e.CPU.ModelName,
		},
		Hostname: e.Hostname,
		Memory: &models.Memory{
			PhysicalBytes: e.Memory.PhysicalBytes,
			UsableBytes:   e.Memory.UsableBytes,
		},
		SystemVendor: &models.SystemVendor{
			Manufacturer: e.SystemVendor.Manufacturer,
			ProductName:  e.SystemVendor.ProductName,
			SerialNumber: e.SystemVendor.SerialNumber,
			Virtual:      e.SystemVendor.Virtual,
		},
	}

	// set manually cpu flags due to missing omitempty in the model
	m.CPU.Flags = make([]string, 0, len(e.CPU.Flags))
	for _, f := range e.CPU.Flags {
		m.CPU.Flags = append(m.CPU.Flags, f)
	}

	m.Disks = make([]*models.Disk, 0, len(e.Disks))
	for _, d := range e.Disks {
		disk := &models.Disk{
			Bootable:  d.Bootable,
			ByID:      d.ByID,
			ByPath:    d.ByPath,
			DriveType: d.DriveType,
			Hctl:      d.Hctl,
			ID:        d.ID,
			IoPerf: &models.IoPerf{
				SyncDuration: d.IoPerf.SyncDuration,
			},
			IsInstallationMedia: d.IsInstallationMedia,
			Model:               d.Model,
			Name:                d.Name,
			Path:                d.Path,
			Serial:              d.Serial,
			SizeBytes:           d.SizeBytes,
			Smart:               d.Smart,
			Vendor:              d.Vendor,
			Wwn:                 d.Wwn,
		}

		m.Disks = append(m.Disks, disk)
	}

	m.Gpus = make([]*models.Gpu, 0, len(e.Gpus))
	for _, g := range e.Gpus {
		gpu := &models.Gpu{
			Address:  g.Address,
			DeviceID: g.DeviceID,
			Name:     g.Name,
			Vendor:   g.Vendor,
			VendorID: g.VendorID,
		}

		m.Gpus = append(m.Gpus, gpu)
	}

	m.HostDevices = make([]*models.HostDevice, 0, len(e.HostDevices))
	for _, h := range e.HostDevices {
		hostDevice := &models.HostDevice{
			DeviceType: h.DeviceType,
			Gid:        h.Gid,
			UID:        h.UID,
			Major:      h.Major,
			Minor:      h.Minor,
			Path:       h.Path,
		}

		m.HostDevices = append(m.HostDevices, hostDevice)
	}

	m.Interfaces = make([]*models.Interface, 0, len(e.Interfaces))
	for _, i := range e.Interfaces {
		ii := &models.Interface{
			Biosdevname:   i.Biosdevname,
			ClientID:      i.ClientID,
			Flags:         i.Flags,
			HasCarrier:    i.HasCarrier,
			IPV4Addresses: i.IPV4Addresses,
			IPV6Addresses: i.IPV6Addresses,
			MacAddress:    i.MacAddress,
			Mtu:           i.Mtu,
			Name:          i.Name,
			Product:       i.Product,
			SpeedMbps:     i.SpeedMbps,
			Vendor:        i.Vendor,
		}

		m.Interfaces = append(m.Interfaces, ii)
	}

	m.Mounts = make([]*models.Mount, 0, len(e.Mounts))
	for _, mm := range e.Mounts {
		mount := &models.Mount{
			Device:    mm.Device,
			Directory: mm.Directory,
			Options:   mm.Options,
			Type:      mm.Type,
		}

		m.Mounts = append(m.Mounts, mount)
	}

	return m
}

func heartbeatEntity2Model(e entity.Heartbeat) models.Heartbeat {
	hardware := hardwareEntity2Model(*e.Hardware)

	m := models.Heartbeat{
		Hardware: &hardware,
		Status:   e.Status.String(),
		Upgrade:  &models.UpgradeStatus{
			// CurrentCommitID:   e.Upgrade.CurrentCommitID,
			// LastUpgradeStatus: e.Upgrade.LastUpgradeStatus,
			// LastUpgradeTime:   e.Upgrade.LastUpgradeTime,
		},
		Version: e.Version,
	}

	m.Workloads = make([]*models.WorkloadStatus, 0, len(e.Workloads))
	for _, w := range e.Workloads {
		ww := models.WorkloadStatus{
			LastDataUpload: strfmt.DateTime(w.LastDataUpload),
			Name:           w.Name,
			Status:         w.Status.String(),
		}

		m.Workloads = append(m.Workloads, &ww)
	}

	m.Events = make([]*models.EventInfo, 0, len(e.Events))
	for _, event := range e.Events {
		me := models.EventInfo{
			Message: event.Message,
			Reason:  event.Reason,
			Type:    event.Type.String(),
		}

		m.Events = append(m.Events, &me)
	}

	return m
}

func configurationModel2Entity(m models.DeviceConfigurationMessage) entity.DeviceConfigurationMessage {
	e := entity.DeviceConfiguration{
		Heartbeat: entity.HeartbeatConfiguration{
			HardwareProfile: entity.HardwareProfileConfiguration{
				Include: m.Configuration.Heartbeat.HardwareProfile.Include,
				Scope:   entity.FullScope,
			},
			Period: time.Duration(int(m.Configuration.Heartbeat.PeriodSeconds) * int(time.Second)),
		},
		Profiles: map[string]map[string]string{},
	}

	for _, p := range m.Configuration.Profiles {
		e.Profiles[p.Name] = make(map[string]string)
		for _, condition := range p.Conditions {
			e.Profiles[p.Name][condition.Name] = condition.Expression
		}
	}

	workloads := make([]entity.Workload, 0, len(m.Workloads))
	for _, w := range m.Workloads {
		podWorkload := entity.PodWorkload{
			Name:          w.Name,
			Namespace:     w.Namespace,
			Annotations:   w.Annotations,
			Configmaps:    w.Configmaps,
			Labels:        w.Labels,
			Specification: w.Specification,
		}

		if w.ImageRegistries != nil {
			podWorkload.ImageRegistryAuth = w.ImageRegistries.AuthFile
		}

		workloads = append(workloads, podWorkload)
	}

	deviceConf := entity.DeviceConfigurationMessage{
		Configuration:               e,
		DeviceID:                    m.DeviceID,
		WorkloadsMonitoringInterval: time.Duration(int(m.WorkloadsMonitoringInterval) * int(time.Second)),
		Workloads:                   workloads,
	}

	return deviceConf
}
