package main

// instance.go
// Single-instance enforcement via PID lockfile.
//
// Lockfile location:
//   Linux/macOS: /tmp/barcodehid.lock
//   Windows:     %TEMP%\barcodehid.lock
//
// Lockfile format (two lines):
//   <pid>
//   <https://ip:port server URL>
//
// On startup: read lockfile → check if PID is alive → if yes, already running.
// On clean exit: remove lockfile.
// On crash: stale lockfile is detected via dead PID check, overwritten.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func lockFilePath() string {
	return filepath.Join(os.TempDir(), "barcodehid.lock")
}

// writeLockFile writes our PID and server URL to the lockfile.
// Called after the server starts successfully.
func writeLockFile(serverURL string) {
	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), serverURL)
	_ = os.WriteFile(lockFilePath(), []byte(content), 0644)
}

// removeLockFile deletes the lockfile on clean shutdown.
func removeLockFile() {
	_ = os.Remove(lockFilePath())
}

// readLockFile parses the lockfile.
// Returns (pid, serverURL, error).
func readLockFile() (int, string, error) {
	data, err := os.ReadFile(lockFilePath())
	if err != nil {
		return 0, "", err
	}
	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(lines) < 1 {
		return 0, "", fmt.Errorf("empty lockfile")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, "", fmt.Errorf("invalid PID in lockfile: %w", err)
	}
	url := ""
	if len(lines) >= 2 {
		url = strings.TrimSpace(lines[1])
	}
	return pid, url, nil
}

// isAlreadyRunning checks whether another instance is running.
// Returns (true, serverURL) if another instance is alive.
// Returns (false, "") if no other instance is running.
func isAlreadyRunning() (bool, string) {
	pid, url, err := readLockFile()
	if err != nil {
		// No lockfile or unreadable — we are the first instance
		return false, ""
	}

	// Check if the PID is still alive using a platform-specific check
	if !isPIDAlive(pid) {
		// Stale lockfile from a crashed/killed process — clean it up
		debugf("stale lockfile (PID %d is dead) — overwriting", pid)
		removeLockFile()
		return false, ""
	}

	// PID is alive and it's not us — another instance is running
	return true, url
}
