// These tests verify monitor events for listeners appearing and disappearing.
package monitor

import (
	"testing"

	"github.com/portpilot/portpilot/internal/ports"
)

// TestDiffDetectsOccupancyAndRelease checks both directions of a snapshot comparison.
func TestDiffDetectsOccupancyAndRelease(t *testing.T) {
	old := []ports.Listener{{Port: 3000, Protocol: "tcp", PID: 10, LocalAddress: "127.0.0.1"}}
	current := []ports.Listener{
		{Port: 3000, Protocol: "tcp", PID: 10, LocalAddress: "127.0.0.1"},
		{Port: 4000, Protocol: "tcp", PID: 20, LocalAddress: "0.0.0.0"},
	}
	changes := Diff(old, current)
	if len(changes) != 1 || !changes[0].Occupied || changes[0].Listener.Port != 4000 {
		t.Fatalf("unexpected changes: %#v", changes)
	}
	changes = Diff(current, nil)
	if len(changes) != 2 || changes[0].Occupied || changes[1].Occupied {
		t.Fatalf("expected release events, got %#v", changes)
	}
}
