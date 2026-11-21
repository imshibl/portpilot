package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/monitor"
	"github.com/portpilot/portpilot/internal/ports"
	"github.com/portpilot/portpilot/internal/processmgmt"
	"github.com/spf13/cobra"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

func newCheckCommand(scanner discovery.Scanner) *cobra.Command {
	return &cobra.Command{
		Use:   "check <port>",
		Short: "Check whether a port is available or occupied",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := parsePort(args[0])
			if err != nil {
				return err
			}
			listeners, err := scanner.List()
			if err != nil {
				return err
			}
			matches := make([]ports.Listener, 0)
			seenProtocols := make(map[string]bool)
			for _, listener := range listeners {
				if listener.Port == port {
					matches = append(matches, listener)
					seenProtocols[listener.Protocol] = true
				}
			}
			for _, protocol := range []string{"tcp", "udp"} {
				if seenProtocols[protocol] {
					continue
				}
				free, err := ports.IsAvailable(port, protocol)
				if err != nil {
					return fmt.Errorf("check %s port %d: %w", protocol, port, err)
				}
				if !free {
					matches = append(matches, ports.Listener{
						Port: port, Protocol: protocol, Process: "unknown",
						LocalAddress: "127.0.0.1",
					})
				}
			}
			if len(matches) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), successStyle.Render(fmt.Sprintf("Port %d is available.", port)))
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), warningStyle.Render(fmt.Sprintf("Port %d is occupied:", port))); err != nil {
				return err
			}
			return printListeners(cmd.OutOrStdout(), matches)
		},
	}
}

func newListCommand(scanner discovery.Scanner) *cobra.Command {
	var asJSON bool
	var processName string
	var rangeValue string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List listening TCP ports",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var portRange *ports.Range
			if rangeValue != "" {
				parsed, err := ports.ParseRange(rangeValue)
				if err != nil {
					return err
				}
				portRange = &parsed
			}
			listeners, err := scanner.List()
			if err != nil {
				return err
			}
			tcpListeners := make([]ports.Listener, 0, len(listeners))
			for _, listener := range listeners {
				if listener.Protocol == "tcp" {
					tcpListeners = append(tcpListeners, listener)
				}
			}
			filtered := ports.Filter(tcpListeners, portRange, processName)
			if asJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(filtered)
			}
			if len(filtered) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No listening TCP ports found.")
				return err
			}
			return printListeners(cmd.OutOrStdout(), filtered)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	cmd.Flags().StringVar(&processName, "process", "", "filter by process name (case-insensitive substring)")
	cmd.Flags().StringVar(&rangeValue, "range", "", "filter by port range, for example 3000-4000")
	return cmd
}

func newKillCommand(scanner discovery.Scanner, killer processmgmt.Killer) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "kill <port>",
		Short: "Terminate the process listening on a port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := parsePort(args[0])
			if err != nil {
				return err
			}
			listeners, err := scanner.List()
			if err != nil {
				return err
			}
			matches := make([]ports.Listener, 0)
			for _, listener := range listeners {
				if listener.Port == port && listener.PID > 0 {
					matches = append(matches, listener)
				}
			}
			if len(matches) == 0 {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "No process found listening on port %d.\n", port)
				return err
			}
			if err := printListeners(cmd.OutOrStdout(), matches); err != nil {
				return err
			}
			reader := bufio.NewReader(cmd.InOrStdin())
			seen := make(map[string]struct{})
			for _, listener := range matches {
				identity := fmt.Sprintf("%d/%d", listener.PID, listener.StartTime)
				if _, exists := seen[identity]; exists {
					continue
				}
				seen[identity] = struct{}{}
				if !force {
					confirmed, err := confirm(reader, cmd.OutOrStdout(), listener)
					if err != nil {
						return err
					}
					if !confirmed {
						if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Skipped PID %d.\n", listener.PID); err != nil {
							return err
						}
						continue
					}
				}
				if err := killer.Kill(listener); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Terminated PID %d (%s).\n", listener.PID, listener.Process); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")
	return cmd
}

func newAvailableCommand() *cobra.Command {
	var rangeValue string
	var count int
	cmd := &cobra.Command{
		Use:   "available",
		Short: "Suggest available TCP ports",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			portRange, err := ports.ParseRange(rangeValue)
			if err != nil {
				return err
			}
			available, err := ports.FindAvailable(portRange, count)
			if err != nil {
				return err
			}
			if len(available) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No available ports found in range %d-%d.\n", portRange.Start, portRange.End)
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Available TCP ports in %d-%d:\n", portRange.Start, portRange.End); err != nil {
				return err
			}
			for _, port := range available {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %d\n", port); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Availability can change as soon as the probe closes.")
			return err
		},
	}
	cmd.Flags().StringVar(&rangeValue, "range", "3000-4000", "port range to check")
	cmd.Flags().IntVar(&count, "count", 5, "maximum number of ports to suggest")
	return cmd
}

func newWatchCommand(scanner discovery.Scanner) *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor listening ports for changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if interval <= 0 {
				return fmt.Errorf("interval must be greater than zero")
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Watching listening ports; press Ctrl+C to stop."); err != nil {
				return err
			}
			previous, err := scanner.List()
			if err != nil {
				return err
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-cmd.Context().Done():
					return nil
				case <-ticker.C:
					current, err := scanner.List()
					if err != nil {
						return err
					}
					for _, change := range monitor.Diff(previous, current) {
						if _, err := fmt.Fprintln(cmd.OutOrStdout(), monitor.Describe(change)); err != nil {
							return err
						}
					}
					previous = current
				}
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", time.Second, "time between scans")
	return cmd
}

func newInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show PortPilot version and platform",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "PortPilot %s\nBuilt with Go for your current platform.\n", version)
			return err
		},
	}
}

func parsePort(value string) (uint32, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: enter a number from 1 to 65535", value)
	}
	return ports.Validate(port)
}

func printListeners(output io.Writer, listeners []ports.Listener) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "PORT\tPROTO\tPROCESS\tPID\tLOCAL ADDRESS\tCOMMAND")
	for _, listener := range listeners {
		processName := listener.Process
		if processName == "" {
			processName = "unknown"
		}
		pid := "-"
		if listener.PID > 0 {
			pid = strconv.FormatInt(int64(listener.PID), 10)
		}
		command := listener.Command
		if command == "" {
			command = "-"
		} else {
			command = strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", " ").Replace(command)
		}
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\n",
			listener.Port, strings.ToUpper(listener.Protocol), processName, pid, listener.LocalAddress, command)
	}
	return writer.Flush()
}

func confirm(input *bufio.Reader, output io.Writer, listener ports.Listener) (bool, error) {
	if _, err := fmt.Fprintf(output, "Terminate PID %d (%s) using %s port %d? [y/N] ",
		listener.PID, listener.Process, strings.ToUpper(listener.Protocol), listener.Port); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
