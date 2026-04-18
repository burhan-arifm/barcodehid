package main

// BarcodeHID
//
// Run modes:
//   ./barcodehid            foreground — logs to terminal, Ctrl+C to stop
//   ./barcodehid --tray     tray mode  — prints QR, detaches on first connect,
//                                        shows system tray icon
//
// Flags:
//   --tray        run as background tray app
//   --port N      server port (default 8765)
//   --host ADDR   bind address (default 0.0.0.0)
//   --no-enter    disable auto-Enter after each scan
//   --debug       verbose logging

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
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

	// Validate embedded assets
	if len(embeddedHTML) == 0 {
		printFail("assets/scanner.html was not embedded at build time.")
		os.Exit(1)
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

	// If this is the daemon child (already detached), go straight to tray
	if os.Getenv("BARCODEHID_DAEMON") == "1" {
		initTray(serverURL, wsURL)
		return
	}

	// Parent process: print QR and wait for first connection
	printQR(serverURL)
	fmt.Println()
	printInfo("Waiting for phone to connect…")
	printInfo("Scan the QR code above with your phone camera")
	printInfo("After connecting, this window can be closed")
	fmt.Println()

	// Wait for first connection with a timeout hint (not a hard limit)
	connected := make(chan struct{}, 1)
	go func() {
		for {
			if gTray.connected.Load() {
				connected <- struct{}{}
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	select {
	case <-connected:
		printOK("Phone connected!")
		printInfo("Detaching to system tray…")
		time.Sleep(500 * time.Millisecond)

		// Detach: on Unix this forks+exits parent; on Windows hides console
		if !detachFromTerminal() {
			// We are the parent — exit, child continues as tray daemon
			os.Exit(0)
		}
		// We are the daemon child — show tray
		initTray(serverURL, wsURL)

	case <-func() chan os.Signal {
		// Also allow Ctrl+C in the terminal during the wait
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		return c
	}():
		fmt.Printf("\nTotal scans this session: %d\n",
			atomic.LoadUint64(&gScanCount))
		log.Println("Bye.")
	}
}
