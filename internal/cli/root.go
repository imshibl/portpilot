// Package cli defines PortPilot's Cobra commands and connects them to core packages.
package cli

import (
	"io"

	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/processmgmt"
	"github.com/spf13/cobra"
)

// version is replaced by GoReleaser for tagged releases.
var version = "0.1.0"

// NewRoot creates the top-level CLI and injects I/O and system services for testability.
func NewRoot(scanner discovery.Scanner, killer processmgmt.Killer, input io.Reader, output, errorOutput io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "portpilot",
		Short:         "Find and manage occupied ports",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(input)
	root.SetOut(output)
	root.SetErr(errorOutput)
	root.AddCommand(
		newCheckCommand(scanner),
		newListCommand(scanner),
		newKillCommand(scanner, killer),
		newAvailableCommand(),
		newWatchCommand(scanner),
		newInfoCommand(),
	)
	return root
}
