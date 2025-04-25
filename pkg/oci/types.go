package oci

import (
	"github.com/oracle/oci-go-sdk/v65/core"
)

type Machine struct {
	Status     string
	InstanceID string
	PublicIP   string
	PrivateIP  string
	Hostname   string
}

func (m Machine) Host() string {
	if m.Hostname != "" {
		return m.Hostname
	}
	if m.PublicIP != "" {
		return m.PublicIP
	}
	return m.PrivateIP
}

// NewMachineFromInstance creates a new Machine struct from an OCI core.Instance struct
func NewMachineFromInstance(instance core.Instance) Machine {
	var publicIP, privateIP, hostname string

	// Get private IP from the instance's VNIC attachments (requires extra API call in OCI, so this is a placeholder)
	// In real implementation, you would fetch the VNIC and get the IPs.

	if instance.DisplayName != nil {
		hostname = *instance.DisplayName
	}
	if instance.Id != nil {
		// Use OCID as InstanceID
	}

	// Status mapping
	status := string(instance.LifecycleState)

	return Machine{
		InstanceID: func() string { if instance.Id != nil { return *instance.Id } else { return "" } }(),
		Hostname:   hostname,
		PrivateIP:  privateIP,
		PublicIP:   publicIP,
		Status:     status,
	}
}
