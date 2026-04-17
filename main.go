package main

// BarcodeHID
//
// Single portable binary per platform. No standard/portable split.
// On Linux: tries uinput first, then falls back through dotool → ydotool → wtype → xdotool.
// On Windows: SendInput Win32 API (no setup needed).
// On macOS: CGEvent Core Graphics (one-time Accessibility permission).
//
// scanner.html is embedded at build time via go:embed.
// beep.mp3 is optionally embedded if assets/beep.mp3 exists at build time (-tags beep).
//
// Build:
//   bash build.sh
//
// Manual:
//   go build -ldflags="-s -w" -trimpath -o barcodehid .
//   go build -tags beep -ldflags="-s -w" -trimpath -o barcodehid .

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

// buildVariant is injected at compile time via -ldflags="-X main.buildVariant=..."
// Used only for the banner display.
var buildVariant = "portable"

var (
	flagPort    = flag.Int("port", 8765, "Server port (default 8765)")
	flagHost    = flag.String("host", "0.0.0.0", "Bind address (default all interfaces)")
	flagNoEnter = flag.Bool("no-enter", false, "Disable auto-Enter after each scan")
	flagDebug   = flag.Bool("debug", false, "Enable verbose debug logging")
)

func main() {
	flag.Parse()

	if *flagDebug {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(log.Ltime)
	}

	// Resolve working directory from binary location.
	// Falls back to cwd when running via `go run .` during development.
	exe, err := os.Executable()
	dir := "."
	if err == nil {
		dir = filepath.Dir(exe)
	}

	// Validate embedded scanner HTML
	if len(embeddedHTML) == 0 {
		printFail("assets/scanner.html was not embedded at build time.")
		printFail("Ensure assets/scanner.html exists and run: go build .")
		os.Exit(1)
	}
	printOK(fmt.Sprintf("scanner.html embedded (%d KB)", len(embeddedHTML)/1024))

	if len(embeddedBeep) > 0 {
		printOK(fmt.Sprintf("beep.mp3 embedded (%d KB)", len(embeddedBeep)/1024))
	} else {
		printInfo("No beep.mp3 — phone will use Web Audio synthesis")
	}

	// Initialise keyboard backend (platform-specific)
	gAutoEnter = !*flagNoEnter
	gKB = newKeyboard()
	gHIDMode = gKB.Mode()
	defer gKB.Close()

	// Start HTTPS + WebSocket server
	if err := startServer(dir, *flagPort, *flagHost); err != nil {
		printFail("Failed to start server: " + err.Error())
		os.Exit(1)
	}

	// Wait for Ctrl+C or SIGTERM
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
