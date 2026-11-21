// These tests exercise CLI behavior with fake discovery and termination services.
package cli

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/portpilot/portpilot/internal/ports"
)

type testScanner struct {
	listeners []ports.Listener
	err       error
}

// List returns test data or a configured error instead of querying the host OS.
func (s testScanner) List() ([]ports.Listener, error) {
	return s.listeners, s.err
}

type testKiller struct {
	killed []ports.Listener
}

// Kill records what the CLI asked to terminate without signaling a real process.
func (k *testKiller) Kill(listener ports.Listener) error {
	k.killed = append(k.killed, listener)
	return nil
}

func execute(t *testing.T, scanner testScanner, killer *testKiller, input string, args ...string) (string, error) {
	// Build a fresh root command so each test has isolated flags, output, and input.
	t.Helper()
	var output, errorOutput bytes.Buffer
	root := NewRoot(scanner, killer, strings.NewReader(input), &output, &errorOutput)
	root.SetArgs(args)
	err := root.Execute()
	return output.String() + errorOutput.String(), err
}

func TestListCommandFiltersAndEncodesJSON(t *testing.T) {
	// The JSON list should honor both filters and exclude UDP because list is TCP-only.
	listeners := []ports.Listener{
		{Port: 3001, Protocol: "tcp", Process: "node", PID: 10, LocalAddress: "127.0.0.1"},
		{Port: 3002, Protocol: "tcp", Process: "python", PID: 20, LocalAddress: "127.0.0.1"},
		{Port: 3001, Protocol: "udp", Process: "node", PID: 10, LocalAddress: "127.0.0.1"},
	}
	output, err := execute(t, testScanner{listeners: listeners}, &testKiller{}, "", "list", "--json", "--range", "3000-3010", "--process", "NODE")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(output, `"port": 3001`) != 1 || strings.Contains(output, `"port": 3002`) {
		t.Fatalf("unexpected filtered JSON: %s", output)
	}
}

// TestCheckRejectsInvalidPort verifies malformed port numbers fail before discovery.
func TestCheckRejectsInvalidPort(t *testing.T) {
	_, err := execute(t, testScanner{}, &testKiller{}, "", "check", "65536")
	if err == nil || !strings.Contains(err.Error(), "65535") {
		t.Fatalf("expected port validation error, got %v", err)
	}
}

// TestCheckFindsBoundUDPSocketWhenDiscoveryHasNoOwner tests the bind-probe fallback.
func TestCheckFindsBoundUDPSocketWhenDiscoveryHasNoOwner(t *testing.T) {
	// The fallback bind probe detects a UDP socket even when discovery returns no owner.
	socket, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	port := socket.LocalAddr().(*net.UDPAddr).Port

	output, err := execute(t, testScanner{}, &testKiller{}, "", "check", strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "occupied") || !strings.Contains(output, "UDP") || !strings.Contains(output, "unknown") {
		t.Fatalf("check did not report the bound UDP port: %q", output)
	}
}

// TestKillRequiresConfirmationUnlessForced checks default safety and the explicit bypass flag.
func TestKillRequiresConfirmationUnlessForced(t *testing.T) {
	// Declining leaves the fake process untouched; --force skips the prompt and invokes the killer.
	listener := ports.Listener{
		Port: 3000, Protocol: "tcp", Process: "test-server", PID: 42, LocalAddress: "127.0.0.1", StartTime: 123,
	}
	scanner := testScanner{listeners: []ports.Listener{listener}}
	killer := &testKiller{}
	output, err := execute(t, scanner, killer, "n\n", "kill", "3000")
	if err != nil {
		t.Fatal(err)
	}
	if len(killer.killed) != 0 || !strings.Contains(output, "Skipped PID 42") {
		t.Fatalf("declined kill was not skipped: %q, killed=%v", output, killer.killed)
	}

	output, err = execute(t, scanner, killer, "", "kill", "3000", "--force")
	if err != nil {
		t.Fatal(err)
	}
	if len(killer.killed) != 1 || !strings.Contains(output, "Terminated PID 42") {
		t.Fatalf("forced kill was not invoked: %q, killed=%v", output, killer.killed)
	}
}

// TestKillReadsSeparateConfirmationsForMultipleProcesses ensures each owner gets its own answer.
func TestKillReadsSeparateConfirmationsForMultipleProcesses(t *testing.T) {
	listeners := []ports.Listener{
		{Port: 3000, Protocol: "tcp", Process: "first", PID: 41, LocalAddress: "127.0.0.1", StartTime: 101},
		{Port: 3000, Protocol: "tcp", Process: "second", PID: 42, LocalAddress: "127.0.0.1", StartTime: 102},
	}
	killer := &testKiller{}
	_, err := execute(t, testScanner{listeners: listeners}, killer, "y\ny\n", "kill", "3000")
	if err != nil {
		t.Fatal(err)
	}
	if len(killer.killed) != 2 {
		t.Fatalf("expected two confirmed processes to be terminated, got %d", len(killer.killed))
	}
}

// TestListPropagatesDiscoveryErrors ensures OS access problems are not hidden.
func TestListPropagatesDiscoveryErrors(t *testing.T) {
	_, err := execute(t, testScanner{err: errors.New("permission denied")}, &testKiller{}, "", "list")
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected discovery error, got %v", err)
	}
}
