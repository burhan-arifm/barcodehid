//go:build darwin

package main

// daemon_darwin.go
// macOS does not support tray/background mode yet.
// The --tray flag is silently ignored on macOS and the app
// runs in foreground mode instead.
// Full tray support is planned for a future release.

func detachFromTerminal() bool {
	// No-op on macOS — always run in foreground
	return true
}
