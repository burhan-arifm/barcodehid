//go:build linux

package main

// daemon_unix.go
// Terminal detach for Linux only.
// macOS tray mode is not supported — the binary runs foreground-only on macOS.

import (
	"os"
	"os/exec"
	"syscall"
)

func detachFromTerminal() bool {
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
	return false
}
