// Package discovery finds local TCP listeners and UDP sockets, including owning process details.
package discovery

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/portpilot/portpilot/internal/ports"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Scanner lets commands use either real operating-system discovery or a test substitute.
type Scanner interface {
	// List returns the sockets currently visible to the running user.
	List() ([]ports.Listener, error)
}

// System uses gopsutil to query the operating system without shelling out to platform commands.
type System struct{}

var (
	// These patterns redact common secret-bearing command-line flags and environment assignments.
	flagSecret = regexp.MustCompile(`(?i)(--?[a-z0-9_-]*(?:token|password|passwd|secret|api[-_]?key|access[-_]?key|private[-_]?key|credential|csrf[-_]?token)[a-z0-9_-]*(?:=|\s+))("[^"]*"|'[^']*'|\S+)`)
	envSecret  = regexp.MustCompile(`(?i)\b([a-z0-9_]*(?:token|password|passwd|secret|api_key|access_key|private_key|credential)[a-z0-9_]*)=([^\s]+)`)
)

// List returns TCP sockets in the listening state and UDP sockets with a local port.
func (System) List() ([]ports.Listener, error) {
	connections, err := gnet.Connections("inet")
	if err != nil {
		return nil, fmt.Errorf("discover listening sockets: %w", err)
	}

	metadata := make(map[int32]processMetadata)
	listeners := make([]ports.Listener, 0, len(connections))
	seen := make(map[string]struct{}, len(connections))
	for _, connection := range connections {
		protocol := ""
		switch connection.Type {
		case 1: // gopsutil reports the operating system's SOCK_STREAM value for TCP.
			if connection.Status != "LISTEN" {
				continue
			}
			protocol = "tcp"
		case 2: // SOCK_DGRAM represents UDP sockets.
			protocol = "udp"
		default:
			continue
		}
		if connection.Laddr.Port == 0 {
			continue
		}
		pid := connection.Pid
		if pid < 0 {
			// A zero PID means the OS exposed the socket but not its owner.
			pid = 0
		}
		// A process may own several sockets, so collect its metadata only once per scan.
		info, ok := metadata[pid]
		if !ok {
			info = inspectProcess(pid)
			metadata[pid] = info
		}
		address := connection.Laddr.IP
		if address == "" {
			address = "*"
		}
		listener := ports.Listener{
			Port:         connection.Laddr.Port,
			Protocol:     protocol,
			Process:      info.name,
			PID:          pid,
			LocalAddress: address,
			Command:      info.command,
			StartTime:    info.startTime,
		}
		key := fmt.Sprintf("%s/%d/%d/%s", protocol, listener.Port, pid, address)
		if _, exists := seen[key]; exists {
			// Some OS APIs report the same socket more than once.
			continue
		}
		seen[key] = struct{}{}
		listeners = append(listeners, listener)
	}
	sort.Slice(listeners, func(i, j int) bool {
		if listeners[i].Port != listeners[j].Port {
			return listeners[i].Port < listeners[j].Port
		}
		if listeners[i].Protocol != listeners[j].Protocol {
			return listeners[i].Protocol < listeners[j].Protocol
		}
		if listeners[i].PID != listeners[j].PID {
			return listeners[i].PID < listeners[j].PID
		}
		return listeners[i].LocalAddress < listeners[j].LocalAddress
	})
	return listeners, nil
}

// processMetadata is the best-effort process information associated with a socket.
type processMetadata struct {
	name      string
	command   string
	startTime int64
}

// inspectProcess reads process details when permitted, preserving the socket if details are hidden.
func inspectProcess(pid int32) processMetadata {
	if pid <= 0 {
		return processMetadata{name: "unknown"}
	}
	result := processMetadata{name: "unknown"}
	proc, err := process.NewProcess(pid)
	if err != nil {
		return result
	}
	if name, err := proc.Name(); err == nil && name != "" {
		result.name = name
	}
	if command, err := proc.Cmdline(); err == nil {
		result.command = redactCommand(command)
	}
	if startTime, err := proc.CreateTime(); err == nil {
		result.startTime = startTime
	}
	return result
}

// redactCommand masks common credentials before command lines reach terminal or JSON output.
func redactCommand(command string) string {
	command = flagSecret.ReplaceAllString(command, `${1}[REDACTED]`)
	return envSecret.ReplaceAllString(command, `${1}=[REDACTED]`)
}
