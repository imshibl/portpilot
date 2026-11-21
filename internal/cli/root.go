package cli

import (
	"io"

	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/processmgmt"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

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
