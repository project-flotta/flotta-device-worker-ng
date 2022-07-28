package entity

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
)

type WorkloadKind string

const (
	PodKind     WorkloadKind = "pod"
	AnsibleKind WorkloadKind = "ansible"
)

type Workload interface {
	Kind() WorkloadKind
	String() string
	Hash() string
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

	// specification
	Specification string
}

func (p PodWorkload) Kind() WorkloadKind {
	return PodKind
}

func (p PodWorkload) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "name: %s\n", p.Name)
	fmt.Fprintf(&sb, "namespace: %s\n", p.Namespace)
	fmt.Fprintf(&sb, "annotations: \n")
	for k, v := range p.Annotations {
		fmt.Fprintf(&sb, "key=%s value=%s\n", k, v)
	}

	fmt.Fprintf(&sb, "secrets: \n")
	for k, v := range p.Secrets {
		fmt.Fprintf(&sb, "key=%s value=%s\n", k, v)
	}

	fmt.Fprintf(&sb, "labels: \n")
	for k, v := range p.Secrets {
		fmt.Fprintf(&sb, "key=%s value=%s\n", k, v)
	}

	fmt.Fprintf(&sb, "configmaps: \n")
	for _, c := range p.Configmaps {
		fmt.Fprintf(&sb, "value=%s\n", c)
	}

	fmt.Fprintf(&sb, "image registries: %s\n", p.ImageRegistryAuth)
	fmt.Fprintf(&sb, "specification: %s\n", p.Specification)

	return sb.String()
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
