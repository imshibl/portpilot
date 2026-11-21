//go:build !windows

package ports

import (
	"errors"
	"syscall"
)

// isAddressInUse recognizes the POSIX error returned when a socket cannot reuse a bound port.
func isAddressInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}
