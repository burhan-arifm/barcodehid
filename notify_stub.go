//go:build !linux

package main

// notify_stub.go
// Notification stubs for Windows and macOS.
// These platforms will get native notifications in a future iteration.
// For now they no-op so the rest of the code compiles unchanged.

import "fmt"

func notify(title, body string) {
	// TODO: Windows — use toast notifications via go-toast
	// TODO: macOS — use osascript display notification
	debugf("notify (stub): %s — %s", title, body)
}

func notifyRunning(ip string, port int) {
	fmt.Printf("  BarcodeHID running at https://%s:%d\n", ip, port)
}

func notifyAlreadyRunning(ip string, port int) {
	fmt.Printf("  BarcodeHID already running at https://%s:%d\n", ip, port)
}
