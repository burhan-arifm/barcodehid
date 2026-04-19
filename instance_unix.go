//go:build linux || darwin

package main

import (
	"os"
	"syscall"
)

// isPIDAlive returns true if the given PID is a running process.
// Uses kill(pid, 0) — sends no signal but checks if the process exists.
func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// kill(pid, 0) returns nil if process exists and we can signal it,
	// ESRCH if it doesn't exist, EPERM if it exists but we can't signal it.
	// Both nil and EPERM mean the process is alive.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true // process exists, we can signal it
	}
	if err == syscall.EPERM {
		return true // process exists, but owned by another user
	}
	return false // ESRCH — no such process
}

// releaseLock removes the lockfile. On Unix we use PID-based locking
// so there's no file descriptor to close.
func releaseLock() {
	removeLockFile()
}
