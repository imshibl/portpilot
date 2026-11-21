package processmgmt

import (
	"fmt"

	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/ports"
	"github.com/shirou/gopsutil/v4/process"
)

type Killer interface {
	Kill(ports.Listener) error
}

type Manager struct {
	scanner discovery.Scanner
}

func New(scanner discovery.Scanner) *Manager {
	return &Manager{scanner: scanner}
}

func (m *Manager) Kill(target ports.Listener) error {
	if target.PID <= 0 || target.StartTime <= 0 {
		return fmt.Errorf("cannot safely identify the owning process for port %d; process metadata is unavailable", target.Port)
	}

	current, err := m.scanner.List()
	if err != nil {
		return fmt.Errorf("recheck port %d before termination: %w", target.Port, err)
	}
	stillOwned := false
	for _, listener := range current {
		if listener.Port == target.Port &&
			listener.Protocol == target.Protocol &&
			listener.PID == target.PID &&
			listener.StartTime == target.StartTime {
			stillOwned = true
			break
		}
	}
	if !stillOwned {
		return fmt.Errorf("process %d no longer owns %s port %d; refusing to terminate it", target.PID, target.Protocol, target.Port)
	}

	proc, err := process.NewProcess(target.PID)
	if err != nil {
		return fmt.Errorf("reopen process %d: %w", target.PID, err)
	}
	startTime, err := proc.CreateTime()
	if err != nil {
		return fmt.Errorf("verify process %d identity: %w", target.PID, err)
	}
	if startTime != target.StartTime {
		return fmt.Errorf("process identity changed for PID %d; refusing to terminate it", target.PID)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("terminate process %d: %w", target.PID, err)
	}
	return nil
}
