// Package ports contains the shared port data types, validation, and filtering logic.
package ports

import (
	"fmt"
	"strconv"
	"strings"
)

// Listener describes one process-owned network socket found on the local machine.
type Listener struct {
	Port         uint32 `json:"port"`              // The local TCP or UDP port number.
	Protocol     string `json:"protocol"`          // "tcp" or "udp".
	Process      string `json:"process"`           // The owning program name, or "unknown".
	PID          int32  `json:"pid"`               // The operating system's process ID.
	LocalAddress string `json:"local_address"`     // The interface address on which the socket is bound.
	Command      string `json:"command,omitempty"` // Redacted command line, when the OS exposes it.
	// StartTime distinguishes the original process from a different process reusing its PID.
	StartTime int64 `json:"-"`
}

// Range is an inclusive range of valid port numbers.
type Range struct {
	Start uint32 // First port included in the range.
	End   uint32 // Last port included in the range.
}

// Validate checks a user-supplied port before it is used in socket operations.
func Validate(port int) (uint32, error) {
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return uint32(port), nil
}

// ParseRange converts text such as "3000-4000" into a validated inclusive range.
func ParseRange(value string) (Range, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return Range{}, fmt.Errorf("invalid port range %q: expected START-END", value)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return Range{}, fmt.Errorf("invalid port range %q: %w", value, err)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return Range{}, fmt.Errorf("invalid port range %q: %w", value, err)
	}
	first, err := Validate(start)
	if err != nil {
		return Range{}, fmt.Errorf("invalid port range %q: %w", value, err)
	}
	last, err := Validate(end)
	if err != nil {
		return Range{}, fmt.Errorf("invalid port range %q: %w", value, err)
	}
	if first > last {
		return Range{}, fmt.Errorf("invalid port range %q: start must not exceed end", value)
	}
	return Range{Start: first, End: last}, nil
}

// Filter returns listeners matching both optional filters; a nil range means "any port".
func Filter(listeners []Listener, portRange *Range, process string) []Listener {
	process = strings.ToLower(process)
	filtered := make([]Listener, 0, len(listeners))
	for _, listener := range listeners {
		if portRange != nil && (listener.Port < portRange.Start || listener.Port > portRange.End) {
			continue
		}
		if process != "" && !strings.Contains(strings.ToLower(listener.Process), process) {
			continue
		}
		filtered = append(filtered, listener)
	}
	return filtered
}
