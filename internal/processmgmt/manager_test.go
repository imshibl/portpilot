// These tests confirm the manager refuses to act when ownership cannot be proven.
package processmgmt

import (
	"testing"

	"github.com/portpilot/portpilot/internal/ports"
)

type testScanner struct {
	listeners []ports.Listener
}

// List returns a fixed snapshot so safety checks can be tested without touching real processes.
func (s testScanner) List() ([]ports.Listener, error) {
	return s.listeners, nil
}

// TestKillRefusesMissingProcessIdentity ensures an unknown owner can never be signaled.
func TestKillRefusesMissingProcessIdentity(t *testing.T) {
	manager := New(testScanner{})
	err := manager.Kill(ports.Listener{Port: 3000, Protocol: "tcp", PID: 0})
	if err == nil {
		t.Fatal("expected process identity error")
	}
}

// TestKillRefusesProcessThatNoLongerOwnsPort verifies the manager honors the latest scan.
func TestKillRefusesProcessThatNoLongerOwnsPort(t *testing.T) {
	// An empty fresh scan means the original owner released the port, so it must not be killed.
	target := ports.Listener{
		Port: 3000, Protocol: "tcp", PID: 1, LocalAddress: "127.0.0.1", StartTime: 123,
	}
	manager := New(testScanner{})
	if err := manager.Kill(target); err == nil {
		t.Fatal("expected stale port ownership error")
	}
}
