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

func main() {
	scanner := discovery.System{}
	root := cli.NewRoot(scanner, processmgmt.New(scanner), os.Stdin, os.Stdout, os.Stderr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root.SetContext(ctx)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
