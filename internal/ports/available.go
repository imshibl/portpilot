package ports

import (
	"fmt"
	"net"
	"strconv"
)

// FindAvailable returns up to count ports that can currently be bound on IPv4 loopback.
// The probe socket is closed before returning, so availability is only a point-in-time result.
func FindAvailable(portRange Range, count int) ([]uint32, error) {
	if portRange.Start < 1 || portRange.End > 65535 || portRange.Start > portRange.End {
		return nil, fmt.Errorf("invalid port range %d-%d", portRange.Start, portRange.End)
	}
	if count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	available := make([]uint32, 0)
	for port := portRange.Start; port <= portRange.End && len(available) < count; port++ {
		free, err := IsAvailable(port, "tcp")
		if err != nil {
			return nil, fmt.Errorf("check TCP port %d: %w", port, err)
		}
		if !free {
			continue
		}
		available = append(available, port)
	}
	return available, nil
}

// IsAvailable briefly binds a loopback socket to test one TCP or UDP port.
// A bind conflict means the port is occupied; other OS errors are returned to the caller.
func IsAvailable(port uint32, protocol string) (bool, error) {
	if port < 1 || port > 65535 {
		return false, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))
	switch protocol {
	case "tcp":
		listener, err := net.Listen("tcp", address)
		if isAddressInUse(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if err := listener.Close(); err != nil {
			return false, err
		}
		return true, nil
	case "udp":
		packetConn, err := net.ListenPacket("udp", address)
		if isAddressInUse(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if err := packetConn.Close(); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported protocol %q", protocol)
	}
}
