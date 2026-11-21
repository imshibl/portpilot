// Package main starts the PortPilot command-line application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/portpilot/portpilot/internal/cli"
	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/processmgmt"
)

// main connects operating-system discovery and process management to the CLI.
func main() {
	scanner := discovery.System{}
	root := cli.NewRoot(scanner, processmgmt.New(scanner), os.Stdin, os.Stdout, os.Stderr)
	// Passing an interrupt-aware context lets long-running commands such as watch exit on Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root.SetContext(ctx)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
