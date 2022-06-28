// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// Heartbeat heartbeat
//
// swagger:model heartbeat
type Heartbeat struct {

	// Events produced by device worker.
	Events []*EventInfo `json:"events"`

	// hardware
	Hardware *HardwareInfo `json:"hardware,omitempty"`

	// status
	// Enum: [up degraded]
	Status string `json:"status,omitempty"`

	// upgrade
	Upgrade *UpgradeStatus `json:"upgrade,omitempty"`

	// version
	Version string `json:"version,omitempty"`

	// workloads
	Workloads []*WorkloadStatus `json:"workloads"`
}

// Validate validates this heartbeat
func (m *Heartbeat) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateEvents(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHardware(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStatus(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUpgrade(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateWorkloads(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Heartbeat) validateEvents(formats strfmt.Registry) error {
	if swag.IsZero(m.Events) { // not required
		return nil
	}

	for i := 0; i < len(m.Events); i++ {
		if swag.IsZero(m.Events[i]) { // not required
			continue
		}

		if m.Events[i] != nil {
			if err := m.Events[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("events" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("events" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *Heartbeat) validateHardware(formats strfmt.Registry) error {
	if swag.IsZero(m.Hardware) { // not required
		return nil
	}

	if m.Hardware != nil {
		if err := m.Hardware.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hardware")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("hardware")
			}
			return err
		}
	}

	return nil
}

var heartbeatTypeStatusPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["up","degraded"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		heartbeatTypeStatusPropEnum = append(heartbeatTypeStatusPropEnum, v)
	}
}

const (

	// HeartbeatStatusUp captures enum value "up"
	HeartbeatStatusUp string = "up"

	// HeartbeatStatusDegraded captures enum value "degraded"
	HeartbeatStatusDegraded string = "degraded"
)

// prop value enum
func (m *Heartbeat) validateStatusEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, heartbeatTypeStatusPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *Heartbeat) validateStatus(formats strfmt.Registry) error {
	if swag.IsZero(m.Status) { // not required
		return nil
	}

	// value enum
	if err := m.validateStatusEnum("status", "body", m.Status); err != nil {
		return err
	}

	return nil
}

func (m *Heartbeat) validateUpgrade(formats strfmt.Registry) error {
	if swag.IsZero(m.Upgrade) { // not required
		return nil
	}

	if m.Upgrade != nil {
		if err := m.Upgrade.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("upgrade")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("upgrade")
			}
			return err
		}
	}

	return nil
}

func (m *Heartbeat) validateWorkloads(formats strfmt.Registry) error {
	if swag.IsZero(m.Workloads) { // not required
		return nil
	}

	for i := 0; i < len(m.Workloads); i++ {
		if swag.IsZero(m.Workloads[i]) { // not required
			continue
		}

		if m.Workloads[i] != nil {
			if err := m.Workloads[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("workloads" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("workloads" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this heartbeat based on the context it is used
func (m *Heartbeat) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateEvents(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateHardware(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUpgrade(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateWorkloads(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Heartbeat) contextValidateEvents(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Events); i++ {

		if m.Events[i] != nil {
			if err := m.Events[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("events" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("events" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *Heartbeat) contextValidateHardware(ctx context.Context, formats strfmt.Registry) error {

	if m.Hardware != nil {
		if err := m.Hardware.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hardware")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("hardware")
			}
			return err
		}
	}

	return nil
}

func (m *Heartbeat) contextValidateUpgrade(ctx context.Context, formats strfmt.Registry) error {

	if m.Upgrade != nil {
		if err := m.Upgrade.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("upgrade")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("upgrade")
			}
			return err
		}
	}

	return nil
}

func (m *Heartbeat) contextValidateWorkloads(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Workloads); i++ {

		if m.Workloads[i] != nil {
			if err := m.Workloads[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("workloads" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("workloads" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *Heartbeat) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Heartbeat) UnmarshalBinary(b []byte) error {
	var res Heartbeat
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
