package monitor

import (
	"fmt"
	"sort"

	"github.com/portpilot/portpilot/internal/ports"
)

type Change struct {
	Occupied bool
	Listener ports.Listener
}

func Diff(previous, current []ports.Listener) []Change {
	before := make(map[string]ports.Listener, len(previous))
	after := make(map[string]ports.Listener, len(current))
	for _, listener := range previous {
		before[key(listener)] = listener
	}
	for _, listener := range current {
		after[key(listener)] = listener
	}
	changes := make([]Change, 0)
	for id, listener := range after {
		if _, ok := before[id]; !ok {
			changes = append(changes, Change{Occupied: true, Listener: listener})
		}
	}
	for id, listener := range before {
		if _, ok := after[id]; !ok {
			changes = append(changes, Change{Listener: listener})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		left, right := changes[i].Listener, changes[j].Listener
		if left.Port != right.Port {
			return left.Port < right.Port
		}
		if left.Protocol != right.Protocol {
			return left.Protocol < right.Protocol
		}
		if left.LocalAddress != right.LocalAddress {
			return left.LocalAddress < right.LocalAddress
		}
		return changes[i].Occupied && !changes[j].Occupied
	})
	return changes
}

func Describe(change Change) string {
	state := "available"
	if change.Occupied {
		state = "occupied"
	}
	return fmt.Sprintf("%s: %s %d on %s (PID %d, %s)", state,
		change.Listener.Protocol, change.Listener.Port, change.Listener.LocalAddress,
		change.Listener.PID, change.Listener.Process)
}

func key(listener ports.Listener) string {
	return fmt.Sprintf("%s/%d/%d/%d/%s", listener.Protocol, listener.Port, listener.PID, listener.StartTime, listener.LocalAddress)
}
