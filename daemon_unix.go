//go:build linux || darwin

package main

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// detachFromTerminal detaches the process from the controlling terminal.
//
// On Linux: re-executes as a detached child (double-fork pattern).
// Returns false in parent (should exit), true in child (should continue).
//
// On macOS: does NOT fork — fyne.io/systray requires the Cocoa run loop
// on the original main thread, which is destroyed by fork+exec.
// Instead we just detach stdin/stdout/stderr and call setsid().
// The terminal will appear to hang briefly then the user can close it —
// or we print a message and the user closes it manually.
// Returns true always on macOS (caller continues as tray).
func detachFromTerminal() bool {
	if runtime.GOOS == "darwin" {
		return detachDarwin()
	}
	return detachLinux()
}

// detachLinux uses the classic re-exec fork pattern.
func detachLinux() bool {
	if os.Getenv("BARCODEHID_DAEMON") == "1" {
		syscall.Setsid() //nolint
		return true
	}

	exe, err := os.Executable()
	if err != nil {
		printWarn("Could not detach: " + err.Error())
		return true
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "BARCODEHID_DAEMON=1")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		printWarn("Could not start daemon: " + err.Error())
		return true
	}

	printOK("BarcodeHID running in background")
	printInfo("Tray icon is now active — right-click to control")
	return false // parent should exit
}

// detachDarwin handles macOS differently.
// fyne.io/systray needs the Cocoa main run loop on the original thread,
// so we cannot fork. Instead:
//   - Redirect stdout/stderr to /dev/null so terminal output stops
//   - Call setsid() to detach from the controlling terminal
//   - Return true so the caller continues on this same process/thread
//
// The terminal may show the shell prompt returning while the app runs
// in the background — this is correct macOS behavior for menu bar apps.
func detachDarwin() bool {
	// Redirect stdout/stderr to /dev/null
	devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err == nil {
		os.Stdout = devNull
		os.Stderr = devNull
		// Redirect fd 1 and 2 at the OS level too
		syscall.Dup2(int(devNull.Fd()), 1) //nolint
		syscall.Dup2(int(devNull.Fd()), 2) //nolint
	}

	// Detach from controlling terminal
	syscall.Setsid() //nolint

	// Always return true on macOS — we never fork, just detach in-place
	return true
}
