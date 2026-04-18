//go:build windows

package main

// autostart_windows.go
// Auto-start on login via Windows registry Run key.
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run\BarcodeHID

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const regKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
const regName = "BarcodeHID"

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(regName)
	return err == nil
}

func enableAutoStart() {
	exe, err := os.Executable()
	if err != nil {
		printWarn("Auto-start: could not get executable path: " + err.Error())
		return
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		printWarn("Auto-start: could not open registry: " + err.Error())
		return
	}
	defer k.Close()

	value := `"` + exe + `" --tray`
	if err := k.SetStringValue(regName, value); err != nil {
		printWarn("Auto-start: could not set registry value: " + err.Error())
		return
	}
	printOK("Auto-start enabled via registry")
}

func disableAutoStart() {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	_ = k.DeleteValue(regName)
	printOK("Auto-start disabled")
}
