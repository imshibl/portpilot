// These tests exercise temporary socket binding without depending on fixed port numbers.
package ports

import (
	"net"
	"testing"
)

// TestFindAvailableSkipsOccupiedPort verifies that an open socket is not suggested.
func TestFindAvailableSkipsOccupiedPort(t *testing.T) {
	// Holding a socket open makes its assigned port unavailable to the probe.
	socket, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	port := uint32(socket.Addr().(*net.TCPAddr).Port)

	available, err := FindAvailable(Range{Start: port, End: port}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(available) != 0 {
		t.Fatalf("occupied port %d was suggested as available", port)
	}
}

// TestFindAvailableRejectsInvalidCount checks that callers must request at least one result.
func TestFindAvailableRejectsInvalidCount(t *testing.T) {
	if _, err := FindAvailable(Range{Start: 3000, End: 3001}, 0); err == nil {
		t.Fatal("expected invalid count error")
	}
}

// TestFindAvailableCapsResultToRange prevents returning more results than candidates.
func TestFindAvailableCapsResultToRange(t *testing.T) {
	// Even a very large requested count cannot return more ports than the range contains.
	available, err := FindAvailable(Range{Start: 65535, End: 65535}, int(^uint(0)>>1))
	if err != nil {
		t.Fatal(err)
	}
	if len(available) > 1 {
		t.Fatalf("returned %d ports from a one-port range", len(available))
	}
}
