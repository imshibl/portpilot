package ports

import (
	"fmt"
	"strconv"
	"strings"
)

type Listener struct {
	Port         uint32 `json:"port"`
	Protocol     string `json:"protocol"`
	Process      string `json:"process"`
	PID          int32  `json:"pid"`
	LocalAddress string `json:"local_address"`
	Command      string `json:"command,omitempty"`
	StartTime    int64  `json:"-"`
}

type Range struct {
	Start uint32
	End   uint32
}

func Validate(port int) (uint32, error) {
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return uint32(port), nil
}

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
