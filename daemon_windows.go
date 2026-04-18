//go:build windows

package main

// daemon_windows.go
// On Windows, "detaching from terminal" means hiding the console window.
// The process keeps running but has no visible window — only the tray icon.
// We use FreeConsole() from kernel32.dll via golang.org/x/sys/windows.

import "golang.org/x/sys/windows"

var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var freeConsole = kernel32.NewProc("FreeConsole")

// detachFromTerminal hides the Windows console window.
// Returns true always — on Windows we don't fork, just hide.
func detachFromTerminal() bool {
	freeConsole.Call()
	return true
}
