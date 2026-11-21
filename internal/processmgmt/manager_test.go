package processmgmt

import (
	"testing"

	"github.com/portpilot/portpilot/internal/ports"
)

type testScanner struct {
	listeners []ports.Listener
}

func (s testScanner) List() ([]ports.Listener, error) {
	return s.listeners, nil
}

func TestKillRefusesMissingProcessIdentity(t *testing.T) {
	manager := New(testScanner{})
	err := manager.Kill(ports.Listener{Port: 3000, Protocol: "tcp", PID: 0})
	if err == nil {
		t.Fatal("expected process identity error")
	}
}

func TestKillRefusesProcessThatNoLongerOwnsPort(t *testing.T) {
	target := ports.Listener{
		Port: 3000, Protocol: "tcp", PID: 1, LocalAddress: "127.0.0.1", StartTime: 123,
	}
	manager := New(testScanner{})
	if err := manager.Kill(target); err == nil {
		t.Fatal("expected stale port ownership error")
	}
}
