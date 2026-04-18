//go:build linux || darwin

package main

// daemon_unix.go
// Detaches the process from the controlling terminal on Linux and macOS.
// Uses the standard double-fork pattern:
//   1. Fork child
//   2. Parent exits (terminal gets its shell back)
//   3. Child calls setsid() — becomes session leader, no controlling terminal
//   4. Child continues running as background daemon

import (
	"os"
	"os/exec"
	"syscall"
)

// detachFromTerminal re-executes the current binary as a detached child.
// The child gets BARCODEHID_DAEMON=1 in its environment so it knows not
// to detach again (avoiding infinite fork loop).
//
// Returns true if we are the daemon child (caller should continue running).
// Returns false if we are the parent (caller should exit).
func detachFromTerminal() bool {
	// Already a daemon — don't fork again
	if os.Getenv("BARCODEHID_DAEMON") == "1" {
		// We are the child: become session leader so we have no
		// controlling terminal. SIGHUP from parent's terminal won't reach us.
		syscall.Setsid() //nolint
		return true
	}

	// We are the parent: re-exec ourselves as a detached child
	exe, err := os.Executable()
	if err != nil {
		printWarn("Could not detach: " + err.Error())
		return true // run in foreground instead
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env    = append(os.Environ(), "BARCODEHID_DAEMON=1")
	cmd.Stdin  = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // new session — detached from terminal
	}

	if err := cmd.Start(); err != nil {
		printWarn("Could not start daemon: " + err.Error())
		return true // run in foreground instead
	}

	// Print PID for the user, then parent exits
	printOK("BarcodeHID running in background")
	printInfo("Tray icon is now active — right-click to control")
	printInfo("To stop: right-click tray icon → Quit")
	return false
}
