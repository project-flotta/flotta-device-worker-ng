package entity

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

type WorkloadKind string

const (
	PodKind     WorkloadKind = "pod"
	AnsibleKind WorkloadKind = "ansible"
)

type Workload interface {
	ID() string
	Kind() WorkloadKind
	String() string
	Hash() string
	Profiles() []WorkloadProfile
}

// PodWorkload represents the workload in form of a pod.
type PodWorkload struct {
	Name string

	// Namespace of the workload
	Namespace string

	// Annotations
	Annotations map[string]string

	// secrets
	Secrets map[string]string

	// configmaps
	Configmaps []string

	// image registries auth file
	ImageRegistryAuth string

	// Workload labels
	Labels map[string]string

	WorkloadProfiles []WorkloadProfile

	// specification
	Specification string
}

func (p PodWorkload) ID() string {
	return p.Name
}

func (p PodWorkload) Kind() WorkloadKind {
	return PodKind
}

func (p PodWorkload) Profiles() []WorkloadProfile {
	return p.WorkloadProfiles
}

func (p PodWorkload) String() string {
	json, err := json.Marshal(p)
	if err != nil {
		return err.Error()
	}

	return string(json)
}

func (p PodWorkload) Hash() string {
	sum := sha256.Sum256(bytes.NewBufferString(p.String()).Bytes())
	return fmt.Sprintf("%x", sum)
}

// AnsibleWorkload represents ansible workload.
type AnsibleWorkload struct {
	Playbook string
}

func (a AnsibleWorkload) Kind() WorkloadKind {
	return AnsibleKind
}

func (a AnsibleWorkload) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "playbook: %s\n", a.Playbook)

	return sb.String()
}

func (a AnsibleWorkload) Hash() string {
	sum := sha256.Sum256(bytes.NewBufferString(a.Playbook).Bytes())
	return fmt.Sprintf("%x", sum)
}

type WorkloadProfile struct {
	Name       string
	Conditions []string
}
