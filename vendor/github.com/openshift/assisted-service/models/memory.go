// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Memory memory
//
// swagger:model memory
type Memory struct {

	// physical bytes
	PhysicalBytes int64 `json:"physical_bytes,omitempty"`

	// The method by which the physical memory was set
	PhysicalBytesMethod MemoryMethod `json:"physical_bytes_method,omitempty"`

	// usable bytes
	UsableBytes int64 `json:"usable_bytes,omitempty"`
}

// Validate validates this memory
func (m *Memory) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePhysicalBytesMethod(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Memory) validatePhysicalBytesMethod(formats strfmt.Registry) error {

	if swag.IsZero(m.PhysicalBytesMethod) { // not required
		return nil
	}

	if err := m.PhysicalBytesMethod.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("physical_bytes_method")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Memory) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Memory) UnmarshalBinary(b []byte) error {
	var res Memory
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
