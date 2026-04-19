package main

// BarcodeHID
//
// Run modes:
//   ./barcodehid              foreground — QR in terminal, Ctrl+C to stop
//   ./barcodehid --tray       tray mode  — startup notification, detaches
//                                          to system tray after first connect
//   ./app.AppImage            → defaults to --tray (via AppRun)
//   ./app.AppImage --foreground → foreground mode from AppImage
//
// Single-instance: if already running, shows notification + opens QR page.

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

var buildVariant = "portable"

var (
	flagPort    = flag.Int("port", 8765, "Server port (default 8765)")
	flagHost    = flag.String("host", "0.0.0.0", "Bind address")
	flagNoEnter = flag.Bool("no-enter", false, "Disable auto-Enter after each scan")
	flagDebug   = flag.Bool("debug", false, "Verbose debug logging")
	flagTray    = flag.Bool("tray", false, "Run as background tray application")
)

func main() {
	flag.Parse()

	if *flagDebug {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(log.Ltime)
	}

	exe, err := os.Executable()
	dir := "."
	if err == nil {
		dir = filepath.Dir(exe)
	}

	if len(embeddedHTML) == 0 {
		printFail("assets/scanner.html was not embedded at build time.")
		os.Exit(1)
	}

	// ── Single-instance check ─────────────────────────────────────────────────
	// Daemon child skips this — parent already checked before forking.
	// The child writes its own PID to the lockfile after forking.
	isDaemon := os.Getenv("BARCODEHID_DAEMON") == "1"

	if !isDaemon {
		running, existingURL := isAlreadyRunning()
		if running {
			ip := getLANIP()
			port := *flagPort

			if existingURL != "" {
				notifyAlreadyRunning(ip, port)
				// Open QR page so user can re-pair if needed
				openQRWindow(existingURL) // openQRWindow appends /qr internally
			} else {
				notify("BarcodeHID is already running", "Right-click the tray icon to show QR code.")
			}
			os.Exit(0)
		}
		// No other instance — write a preliminary lockfile with our PID.
		// This will be overwritten with the server URL once the server starts.
		// Released on exit via defer.
		defer releaseLock()
	}

	if *flagTray {
		runTrayMode(dir)
	} else {
		runForegroundMode(dir)
	}
}

// ── Foreground mode ───────────────────────────────────────────────────────────

func runForegroundMode(dir string) {
	printOK(fmt.Sprintf("scanner.html embedded (%d KB)", len(embeddedHTML)/1024))
	if len(embeddedBeep) > 0 {
		printOK(fmt.Sprintf("beep.mp3 embedded (%d KB)", len(embeddedBeep)/1024))
	} else {
		printInfo("No beep.mp3 — phone will use Web Audio synthesis")
	}

	gAutoEnter = !*flagNoEnter
	gKB = newKeyboard()
	gHIDMode = gKB.Mode()
	defer gKB.Close()

	if err := startServer(dir, *flagPort, *flagHost); err != nil {
		printFail("Server error: " + err.Error())
		os.Exit(1)
	}

	// Write URL to lockfile so a second instance can find us
	ip := getLANIP()
	writeLockFile(fmt.Sprintf("https://%s:%d", ip, *flagPort))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Printf("\nTotal scans this session: %d\n",
		atomic.LoadUint64(&gScanCount))
	gLogMu.Lock()
	if gLogFile != nil {
		gLogFile.Close()
	}
	gLogMu.Unlock()
	log.Println("Bye.")
}

// ── Tray mode ─────────────────────────────────────────────────────────────────

func runTrayMode(dir string) {
	gAutoEnter = !*flagNoEnter
	gKB = newKeyboard()
	gHIDMode = gKB.Mode()
	defer gKB.Close()

	if err := startServer(dir, *flagPort, *flagHost); err != nil {
		printFail("Server error: " + err.Error())
		os.Exit(1)
	}

	ip := getLANIP()
	port := *flagPort
	serverURL := fmt.Sprintf("https://%s:%d", ip, port)
	wsURL := fmt.Sprintf("wss://%s:%d", ip, port)

	// Write URL to lockfile so a second instance can find us
	writeLockFile(serverURL)

	// Detach immediately — no waiting for phone connection.
	// The tray icon + notification give the user everything they need.
	if os.Getenv("BARCODEHID_DAEMON") == "1" {
		// We are the daemon child — overwrite lockfile with our PID,
		// send startup notification, then run the tray event loop.
		writeLockFile(serverURL)
		notifyRunning(ip, port)
		initTray(serverURL, wsURL)
		releaseLock()
		return
	}

	// We are the parent — fork to background immediately.
	if !detachFromTerminal() {
		// Parent: print a brief message then exit.
		// Child continues as the background daemon.
		printOK("BarcodeHID started in background")
		printInfo("Check your system tray for the tray icon")
		printInfo("Right-click tray → Show QR to pair your phone")
		os.Exit(0)
	}
	// Child reaches here — handled above in the BARCODEHID_DAEMON branch
	// after re-exec, so this point is never reached.
}
