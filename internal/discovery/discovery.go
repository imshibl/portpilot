package discovery

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/portpilot/portpilot/internal/ports"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Scanner interface {
	List() ([]ports.Listener, error)
}

type System struct{}

var (
	flagSecret = regexp.MustCompile(`(?i)(--?[a-z0-9_-]*(?:token|password|passwd|secret|api[-_]?key|access[-_]?key|private[-_]?key|credential|csrf[-_]?token)[a-z0-9_-]*(?:=|\s+))("[^"]*"|'[^']*'|\S+)`)
	envSecret  = regexp.MustCompile(`(?i)\b([a-z0-9_]*(?:token|password|passwd|secret|api_key|access_key|private_key|credential)[a-z0-9_]*)=([^\s]+)`)
)

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
		case 1:
			if connection.Status != "LISTEN" {
				continue
			}
			protocol = "tcp"
		case 2:
			protocol = "udp"
		default:
			continue
		}
		if connection.Laddr.Port == 0 {
			continue
		}
		pid := connection.Pid
		if pid < 0 {
			pid = 0
		}
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

type processMetadata struct {
	name      string
	command   string
	startTime int64
}

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

func redactCommand(command string) string {
	command = flagSecret.ReplaceAllString(command, `${1}[REDACTED]`)
	return envSecret.ReplaceAllString(command, `${1}=[REDACTED]`)
}
