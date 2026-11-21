package ports

import (
	"net"
	"testing"
)

func TestFindAvailableSkipsOccupiedPort(t *testing.T) {
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

func TestFindAvailableRejectsInvalidCount(t *testing.T) {
	if _, err := FindAvailable(Range{Start: 3000, End: 3001}, 0); err == nil {
		t.Fatal("expected invalid count error")
	}
}

func TestFindAvailableCapsResultToRange(t *testing.T) {
	available, err := FindAvailable(Range{Start: 65535, End: 65535}, int(^uint(0)>>1))
	if err != nil {
		t.Fatal(err)
	}
	if len(available) > 1 {
		t.Fatalf("returned %d ports from a one-port range", len(available))
	}
}
