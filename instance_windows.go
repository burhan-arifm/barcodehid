//go:build windows

package main

import (
	"golang.org/x/sys/windows"
)

// isPIDAlive returns true if the given PID is a running process on Windows.
// Opens a handle with SYNCHRONIZE access — succeeds if process exists.
func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false // process doesn't exist or access denied (treat as dead)
	}
	windows.CloseHandle(handle)
	return true
}

// releaseLock removes the lockfile on Windows.
func releaseLock() {
	removeLockFile()
}
