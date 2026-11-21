//go:build windows

package ports

import (
	"testing"

	"golang.org/x/sys/windows"
)

// TestWindowsAddressInUseMapping checks Winsock's raw error value and does not hide access errors.
func TestWindowsAddressInUseMapping(t *testing.T) {
	if !isAddressInUse(windows.WSAEADDRINUSE) {
		t.Fatal("WSAEADDRINUSE was not recognized")
	}
	if isAddressInUse(windows.WSAEACCES) {
		t.Fatal("WSAEACCES was incorrectly treated as an occupied port")
	}
}
