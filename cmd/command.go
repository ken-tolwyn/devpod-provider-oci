package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/devpod-provider-oci/pkg/oci"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ssh"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/spf13/cobra"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Command an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			ociProvider, err := oci.NewProvider(context.Background(), true, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(
				context.Background(),
				ociProvider,
				provider.FromEnvironment(),
				log.Default,
			)
		},
	}

	return commandCmd
}

// Run runs the command logic
func (cmd *CommandCmd) Run(
	ctx context.Context,
	providerOci *oci.OciProvider,
	machine *provider.Machine,
	logs log.Logger,
) error {
	command := os.Getenv("COMMAND")
	if command == "" {
		return fmt.Errorf("command environment variable is missing")
	}

	// get private key
	privateKey, err := ssh.GetPrivateKeyRawBase(providerOci.Config.MachineFolder)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	// get instance
	instance, err := oci.GetDevpodRunningInstance(
		ctx,
		providerOci,
		providerOci.Config.MachineID,
	)
	if err != nil {
		return err
	} else if instance.Status == "" {
		return fmt.Errorf("instance %s doesn't exist", providerOci.Config.MachineID)
	}

	host := instance.Host()
	sshClient, err := ssh.NewSSHClient("devpod", host+":22", privateKey)
	if err != nil {
		logs.Debugf("error connecting to ip [%s]: %v", host, err)
		return err
	} else {
		// successfully connected to the public ip
		defer sshClient.Close()
		return ssh.Run(ctx, sshClient, command, os.Stdin, os.Stdout, os.Stderr)
	}
}

func waitForPort(ctx context.Context, addr string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			l, err := net.Listen("tcp", addr)
			if err != nil {
				// port is taken
				return
			}
			_ = l.Close()
			time.Sleep(1 * time.Second)
		}
	}

}
func findAvailablePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return -1, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
