//go:build darwin

package main

// autostart_darwin.go
// Auto-start on login via launchd LaunchAgent.
// Creates/removes ~/Library/LaunchAgents/io.barcodehid.plist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "io.barcodehid.plist")
}

func isAutoStartEnabled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

func enableAutoStart() {
	exe, err := os.Executable()
	if err != nil {
		printWarn("Auto-start: could not get executable path: " + err.Error())
		return
	}

	path := plistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		printWarn("Auto-start: could not create LaunchAgents dir: " + err.Error())
		return
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.barcodehid</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>--tray</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <false/>
</dict>
</plist>
`, exe)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		printWarn("Auto-start: could not write plist: " + err.Error())
		return
	}

	// Load immediately so it takes effect without reboot
	_ = exec.Command("launchctl", "load", path).Run()
	printOK("Auto-start enabled: " + path)
}

func disableAutoStart() {
	path := plistPath()
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		printWarn("Auto-start: could not remove plist: " + err.Error())
		return
	}
	printOK("Auto-start disabled")
}
