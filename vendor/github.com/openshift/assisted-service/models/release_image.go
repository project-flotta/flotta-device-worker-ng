// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// ReleaseImage release image
//
// swagger:model release-image
type ReleaseImage struct {

	// The CPU architecture of the image (x86_64/arm64/etc).
	// Required: true
	CPUArchitecture *string `json:"cpu_architecture" gorm:"default:'x86_64'"`

	// Version of the OpenShift cluster.
	// Required: true
	OpenshiftVersion *string `json:"openshift_version"`

	// The installation image of the OpenShift cluster.
	// Required: true
	URL *string `json:"url"`

	// OCP version from the release metadata.
	// Required: true
	Version *string `json:"version"`
}

// Validate validates this release image
func (m *ReleaseImage) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCPUArchitecture(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpenshiftVersion(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateURL(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVersion(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ReleaseImage) validateCPUArchitecture(formats strfmt.Registry) error {

	if err := validate.Required("cpu_architecture", "body", m.CPUArchitecture); err != nil {
		return err
	}

	return nil
}

func (m *ReleaseImage) validateOpenshiftVersion(formats strfmt.Registry) error {

	if err := validate.Required("openshift_version", "body", m.OpenshiftVersion); err != nil {
		return err
	}

	return nil
}

func (m *ReleaseImage) validateURL(formats strfmt.Registry) error {

	if err := validate.Required("url", "body", m.URL); err != nil {
		return err
	}

	return nil
}

func (m *ReleaseImage) validateVersion(formats strfmt.Registry) error {

	if err := validate.Required("version", "body", m.Version); err != nil {
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ReleaseImage) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ReleaseImage) UnmarshalBinary(b []byte) error {
	var res ReleaseImage
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
