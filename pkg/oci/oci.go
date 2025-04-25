package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"

	"pkg/oci"
)

type OciProvider struct {
	Config            *Options
	OciConfigProvider common.ConfigurationProvider
	Log               log.Logger
	WorkingDirectory  string
}

func NewProvider(ctx context.Context, withFolder bool, logs log.Logger) (*OciProvider, error) {
	config, err := FromEnv(withFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to load OCI provider config: %w", err)
	}

	ociConfigProvider := common.NewRawConfigurationProvider(
		config.TenancyOCID,
		config.UserOCID,
		config.Region,
		config.Fingerprint,
		config.PrivateKeyPath,
		nil, // Passphrase, if needed
	)

	provider := &OciProvider{
		Config:            config,
		OciConfigProvider: ociConfigProvider,
		Log:               logs,
	}

	return provider, nil
}

func Create(
	ctx context.Context,
	provider *OciProvider,
) (Machine, error) {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return Machine{}, fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}

	// Prepare launch details
	launchDetails := core.LaunchInstanceDetails{
		CompartmentId: &provider.Config.CompartmentOCID,
		DisplayName:   &provider.Config.MachineID,
		ImageId:       &provider.Config.ImageOCID,
		Shape:         &provider.Config.Shape,
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId:      &provider.Config.SubnetOCID,
			AssignPublicIp: common.Bool(true),
		},
		Metadata: map[string]interface{}{
			"ssh_authorized_keys": provider.Config.SSHPublicKey,
		},
	}

	launchReq := core.LaunchInstanceRequest{
		LaunchInstanceDetails: launchDetails,
	}

	launchResp, err := computeClient.LaunchInstance(ctx, launchReq)
	if err != nil {
		return Machine{}, fmt.Errorf("failed to launch OCI instance: %w", err)
	}

	instanceID := launchResp.Instance.Id

	// Wait for instance to be RUNNING
	getReq := core.GetInstanceRequest{
		InstanceId: instanceID,
	}
	for {
		getResp, err := computeClient.GetInstance(ctx, getReq)
		if err != nil {
			return Machine{}, fmt.Errorf("failed to get OCI instance: %w", err)
		}
		if getResp.Instance.LifecycleState == core.InstanceLifecycleStateRunning {
			return NewMachineFromInstance(getResp.Instance), nil
		}
		// Wait and poll again
		select {
		case <-ctx.Done():
			return Machine{}, ctx.Err()
		default:
			// Sleep for a few seconds before polling again
			// (in production, use a proper waiter)
			time.Sleep(5 * time.Second)
		}
	}
}

func Start(ctx context.Context, provider *OciProvider, instanceID string) error {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}
	actionReq := core.InstanceActionRequest{
		InstanceId: &instanceID,
		Action:     core.InstanceActionActionStart,
	}
	_, err = computeClient.InstanceAction(ctx, actionReq)
	if err != nil {
		return fmt.Errorf("failed to start OCI instance: %w", err)
	}
	return nil
}

func Stop(ctx context.Context, provider *OciProvider, instanceID string) error {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}
	actionReq := core.InstanceActionRequest{
		InstanceId: &instanceID,
		Action:     core.InstanceActionActionStop,
	}
	_, err = computeClient.InstanceAction(ctx, actionReq)
	if err != nil {
		return fmt.Errorf("failed to stop OCI instance: %w", err)
	}
	return nil
}

func Status(ctx context.Context, provider *OciProvider, instanceID string) (client.Status, error) {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}
	getReq := core.GetInstanceRequest{
		InstanceId: &instanceID,
	}
	getResp, err := computeClient.GetInstance(ctx, getReq)
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("failed to get OCI instance: %w", err)
	}
	switch getResp.Instance.LifecycleState {
	case core.InstanceLifecycleStateRunning:
		return client.StatusRunning, nil
	case core.InstanceLifecycleStateStopped:
		return client.StatusStopped, nil
	case core.InstanceLifecycleStateTerminated:
		return client.StatusNotFound, nil
	default:
		return client.StatusBusy, nil
	}
}

func Delete(ctx context.Context, provider *OciProvider, instanceID string) error {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}
	terminateReq := core.TerminateInstanceRequest{
		InstanceId: &instanceID,
	}
	_, err = computeClient.TerminateInstance(ctx, terminateReq)
	if err != nil {
		return fmt.Errorf("failed to terminate OCI instance: %w", err)
	}
	return nil
}

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"

	"pkg/oci"
)

// GetDevpodRunningInstance finds a running OCI instance by display name (MachineID)
func GetDevpodRunningInstance(
	ctx context.Context,
	provider *OciProvider,
	machineID string,
) (Machine, error) {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider.OciConfigProvider)
	if err != nil {
		return Machine{}, fmt.Errorf("failed to create OCI ComputeClient: %w", err)
	}

	// List instances in the compartment, filter by display name and running state
	listReq := core.ListInstancesRequest{
		CompartmentId: &provider.Config.CompartmentOCID,
		DisplayName:   &machineID,
		LifecycleState: core.InstanceLifecycleStateRunning,
	}

	resp, err := computeClient.ListInstances(ctx, listReq)
	if err != nil {
		return Machine{}, fmt.Errorf("failed to list OCI instances: %w", err)
	}
	if len(resp.Items) == 0 {
		return Machine{}, nil
	}

	// Use the first matching instance
	instance := resp.Items[0]
	return NewMachineFromInstance(instance), nil
}
