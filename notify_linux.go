//go:build linux

package main

// notify_linux.go
// Sends desktop notifications via notify-send (libnotify).
// Works on any freedesktop.org-compliant DE: KDE, GNOME, XFCE, etc.
// Falls back silently if notify-send is not installed.

import (
	"fmt"
	"os/exec"
)

// notify sends a desktop notification.
// title and body are plain text — no markup, fully DE agnostic.
func notify(title, body string) {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		// notify-send not installed — skip silently
		debugf("notify-send not found, skipping notification")
		return
	}

	args := []string{
		"--app-name=BarcodeHID",
		"--urgency=normal",
		"--expire-time=6000", // 6 seconds
		"--icon=input-keyboard",
		title,
		body,
	}

	if err := exec.Command(path, args...).Run(); err != nil {
		debugf("notify-send failed: %v", err)
	}
}

// notifyRunning sends the startup notification with the server URL.
func notifyRunning(ip string, port int) {
	notify(
		"BarcodeHID is running",
		fmt.Sprintf("Open on your phone:\nhttps://%s:%d\n\nRight-click tray icon to show QR code.", ip, port),
	)
}

// notifyAlreadyRunning sends a notification when a second instance starts.
func notifyAlreadyRunning(ip string, port int) {
	notify(
		"BarcodeHID is already running",
		fmt.Sprintf("Server is at:\nhttps://%s:%d\n\nOpening QR code page…", ip, port),
	)
}
