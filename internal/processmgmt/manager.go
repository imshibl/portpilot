// Package processmgmt contains the safety checks and OS call used to stop a port-owning process.
package processmgmt

import (
	"fmt"

	"github.com/portpilot/portpilot/internal/discovery"
	"github.com/portpilot/portpilot/internal/ports"
	"github.com/shirou/gopsutil/v4/process"
)

// Killer is the operation the CLI calls after the user confirms termination.
type Killer interface {
	Kill(ports.Listener) error
}

// Manager rechecks socket ownership before asking the OS to terminate a process.
type Manager struct {
	scanner discovery.Scanner
}

// New creates a manager that uses scanner to verify port ownership before killing.
func New(scanner discovery.Scanner) *Manager {
	return &Manager{scanner: scanner}
}

// Kill stops only the process identity that was previously discovered for target's port.
func (m *Manager) Kill(target ports.Listener) error {
	if target.PID <= 0 || target.StartTime <= 0 {
		// Without both values, a PID could refer to an unrelated or already-replaced process.
		return fmt.Errorf("cannot safely identify the owning process for port %d; process metadata is unavailable", target.Port)
	}

	// Re-scan immediately before termination so a process that released the port is not killed.
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
	// gopsutil maps this operation to the current platform's process termination API.
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("terminate process %d: %w", target.PID, err)
	}
	return nil
}
