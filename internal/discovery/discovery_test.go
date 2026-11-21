package discovery

import (
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestListFindsCurrentProcessListener(t *testing.T) {
	socket, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	port := uint32(socket.Addr().(*net.TCPAddr).Port)

	listeners, err := (System{}).List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	for _, listener := range listeners {
		if listener.Port == port && listener.Protocol == "tcp" && listener.PID == int32(os.Getpid()) {
			return
		}
	}
	t.Fatalf("did not find current process listening on TCP port %s", strconv.Itoa(int(port)))
}

func TestRedactCommandSecrets(t *testing.T) {
	command := "server --csrf_token abc123 --api-key=def456 PASSWORD=hunter2 --port 3000"
	got := redactCommand(command)
	for _, secret := range []string{"abc123", "def456", "hunter2"} {
		if strings.Contains(got, secret) {
			t.Errorf("redactCommand() leaked %q in %q", secret, got)
		}
	}
	for _, expected := range []string{"--csrf_token [REDACTED]", "--api-key=[REDACTED]", "PASSWORD=[REDACTED]", "--port 3000"} {
		if !strings.Contains(got, expected) {
			t.Errorf("redactCommand() = %q, missing %q", got, expected)
		}
	}
}
