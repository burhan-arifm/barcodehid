package main

// tray.go
// System tray integration and background mode logic.
//
// --tray flow:
//   1. Start server in foreground, print QR to terminal
//   2. Wait for first phone connection
//   3. Fork/detach from terminal, show tray icon
//   4. Tray menu: Show QR | Copy URL | Auto-start | Quit
//
// On subsequent --tray runs (phone already paired):
//   1. Start server
//   2. Go straight to tray without waiting for connection
//      (phone will auto-reconnect from saved URL)

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"

	"fyne.io/systray"
)

// trayState holds mutable state updated from the WebSocket handler
type trayState struct {
	connected atomic.Bool
	scanCount uint64 // same as gScanCount, aliased for clarity
	serverURL string // https:// URL for QR
	wsURL     string // wss:// URL for display/copy
}

var gTray trayState

// initTray is called after the server starts.
// It blocks until systray exits (user clicks Quit).
func initTray(serverURL, wsURL string) {
	gTray.serverURL = serverURL
	gTray.wsURL = wsURL
	systray.Run(onTrayReady, onTrayExit)
}

func onTrayReady() {
	systray.SetIcon(embeddedTrayIcon)
	systray.SetTitle("BarcodeHID")
	systray.SetTooltip("BarcodeHID — Barcode scanner")

	// Menu items
	mStatus := systray.AddMenuItem("Not connected", "")
	mStatus.Disable()

	systray.AddSeparator()

	mShowQR := systray.AddMenuItem("Show QR Code", "Open QR pairing window")
	mCopyURL := systray.AddMenuItem("Copy wss:// URL", "Copy WebSocket URL to clipboard")

	systray.AddSeparator()

	mAutoStart := systray.AddMenuItemCheckbox(
		"Auto-start on login", "Start BarcodeHID automatically at login",
		isAutoStartEnabled(),
	)

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit BarcodeHID", "Stop the server and exit")

	// Status update loop
	go func() {
		var lastCount uint64
		var lastConn bool
		for {
			count := atomic.LoadUint64(&gScanCount)
			conn := gTray.connected.Load()

			if count != lastCount || conn != lastConn {
				lastCount = count
				lastConn = conn
				if conn {
					mStatus.SetTitle(fmt.Sprintf("Connected — %d scan(s)", count))
				} else {
					mStatus.SetTitle("Not connected")
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Event loop
	go func() {
		for {
			select {
			case <-mShowQR.ClickedCh:
				openQRWindow(gTray.serverURL)

			case <-mCopyURL.ClickedCh:
				copyToClipboard(gTray.wsURL)

			case <-mAutoStart.ClickedCh:
				if mAutoStart.Checked() {
					mAutoStart.Uncheck()
					disableAutoStart()
				} else {
					mAutoStart.Check()
					enableAutoStart()
				}

			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onTrayExit() {
	// Clean shutdown — main() will exit after systray.Run returns
}

// openQRWindow opens the /qr page in the default browser.
// This is the simplest cross-platform approach — no native window needed.
// The /qr endpoint serves a clean full-screen QR page.
func openQRWindow(serverURL string) {
	qrURL := serverURL + "/qr"
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", qrURL).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", qrURL).Start()
	case "darwin":
		err = exec.Command("open", qrURL).Start()
	}
	if err != nil {
		debugf("openQRWindow: %v", err)
	}
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		// Try wl-copy (Wayland) then xclip (X11)
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy", text)
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = stringReader(text)
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
			cmd.Stdin = stringReader(text)
		}
	case "darwin":
		cmd = exec.Command("pbcopy")
		cmd.Stdin = stringReader(text)
	case "windows":
		cmd = exec.Command("clip")
		cmd.Stdin = stringReader(text)
	}
	if cmd != nil {
		if err := cmd.Run(); err != nil {
			debugf("copyToClipboard: %v", err)
		}
	}
}

// stringReader returns a reader for a string (for stdin piping)
type strReader struct {
	s   string
	pos int
}

func (r *strReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.s) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}
func stringReader(s string) *strReader { return &strReader{s: s} }

// validateServerURL sanity-checks the URL before using it
func validateServerURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Host != ""
}
