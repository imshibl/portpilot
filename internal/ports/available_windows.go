//go:build windows

package ports

import (
	"errors"

	"golang.org/x/sys/windows"
)

// Windows reports Winsock error numbers, which differ from syscall.EADDRINUSE's POSIX value.
func isAddressInUse(err error) bool {
	return errors.Is(err, windows.WSAEADDRINUSE)
}
