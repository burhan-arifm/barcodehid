//go:build linux

package main

// autostart_linux.go
// Auto-start on login via XDG autostart spec.
// Works on GNOME, KDE, XFCE, and any XDG-compliant desktop environment.
// Creates/removes ~/.config/autostart/barcodehid.desktop

import (
	"fmt"
	"os"
	"path/filepath"
)

func autostartPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "autostart", "barcodehid.desktop")
}

func isAutoStartEnabled() bool {
	_, err := os.Stat(autostartPath())
	return err == nil
}

func enableAutoStart() {
	exe, err := os.Executable()
	if err != nil {
		printWarn("Auto-start: could not get executable path: " + err.Error())
		return
	}

	path := autostartPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		printWarn("Auto-start: could not create autostart dir: " + err.Error())
		return
	}

	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Type=Application
Name=BarcodeHID
Comment=Barcode scanner — scan with phone, type on PC
Exec=%s --tray
Icon=input-keyboard
Terminal=false
Categories=Utility;
X-GNOME-Autostart-enabled=true
`, exe)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		printWarn("Auto-start: could not write desktop file: " + err.Error())
		return
	}
	printOK("Auto-start enabled: " + path)
}

func disableAutoStart() {
	path := autostartPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		printWarn("Auto-start: could not remove desktop file: " + err.Error())
		return
	}
	printOK("Auto-start disabled")
}
