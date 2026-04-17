package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
)

// ── Global server state ───────────────────────────────────────────────────────

var (
	gKB        Keyboard
	gAutoEnter bool
	gHIDMode   string
	gScanCount uint64
	gPortable  bool

	gLogFile *os.File
	gLogMu   sync.Mutex

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// ── WebSocket protocol ────────────────────────────────────────────────────────

type wsIn struct {
	Type      string `json:"type"`
	Value     string `json:"value,omitempty"`
	SendEnter *bool  `json:"send_enter,omitempty"`
}

type wsOut struct {
	Type          string `json:"type"`
	AutoEnter     bool   `json:"auto_enter,omitempty"`
	HIDMode       string `json:"hid_mode,omitempty"`
	HasBeep       bool   `json:"has_beep,omitempty"`
	ServerVersion string `json:"server_version,omitempty"`
	Value         string `json:"value,omitempty"`
	Count         uint64 `json:"count,omitempty"`
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// WebSocket upgrade
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		wsHandler(w, r)
		return
	}

	debugf("HTTPS GET %s from %s", r.URL.Path, r.RemoteAddr)

	switch r.URL.Path {
	case "/", "/index.html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(embeddedHTML)

	case "/beep.mp3":
		if len(embeddedBeep) > 0 {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = w.Write(embeddedBeep)
		} else {
			http.NotFound(w, r)
		}

	default:
		http.NotFound(w, r)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logf("WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	addr := conn.RemoteAddr().String()
	logf("📱  Phone connected from %s", addr)

	// Tell phone about server config so UI can sync
	_ = conn.WriteJSON(wsOut{
		Type:          "config",
		AutoEnter:     gAutoEnter,
		HIDMode:       gHIDMode,
		HasBeep:       len(embeddedBeep) > 0,
		ServerVersion: "2.0.0",
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure) {
				logf("WS error from %s: %v", addr, err)
			}
			break
		}

		var msg wsIn
		if err := json.Unmarshal(raw, &msg); err != nil {
			logf("Bad JSON from %s: %v", addr, err)
			continue
		}

		switch msg.Type {

		case "scan":
			value := strings.TrimSpace(msg.Value)
			if value == "" {
				continue
			}
			sendEnter := gAutoEnter
			if msg.SendEnter != nil {
				sendEnter = *msg.SendEnter
			}

			count := atomic.AddUint64(&gScanCount, 1)
			enterLabel := "value-only"
			if sendEnter {
				enterLabel = "+ENTER"
			}
			logf("#%04d  [%s]  %s  (from %s)", count, enterLabel, value, addr)
			writeScanLog(value, addr, sendEnter, count)

			// Type in a goroutine so the WS loop stays responsive
			go func(v string, enter bool) {
				time.Sleep(50 * time.Millisecond)
				gKB.TypeString(v)
				if enter {
					gKB.PressEnter()
				}
			}(value, sendEnter)

			_ = conn.WriteJSON(wsOut{Type: "ack", Value: value, Count: count})

		case "ping":
			_ = conn.WriteJSON(wsOut{Type: "pong"})

		default:
			debugf("Unknown WS message type: %q", msg.Type)
		}
	}

	logf("📵  Phone disconnected (%s)", addr)
}

// ── Scan log ──────────────────────────────────────────────────────────────────

func openScanLog(dir string) {
	path := filepath.Join(dir, "scans.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		printWarn("Could not open scan log: " + err.Error())
		return
	}
	gLogFile = f
}

func writeScanLog(value, addr string, enter bool, count uint64) {
	gLogMu.Lock()
	defer gLogMu.Unlock()
	if gLogFile == nil {
		return
	}
	label := "value-only"
	if enter {
		label = "+ENTER"
	}
	fmt.Fprintf(gLogFile, "[%s] #%04d  [%s]  %s  (from %s)\n",
		time.Now().Format("2006-01-02 15:04:05"), count, label, value, addr)
}

// ── Network ───────────────────────────────────────────────────────────────────

func getLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// ── QR code ───────────────────────────────────────────────────────────────────

func printQR(url string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		printWarn("QR generation failed: " + err.Error())
		return
	}
	bmp  := q.Bitmap()
	rows := len(bmp)
	if rows == 0 {
		return
	}
	cols := len(bmp[0])

	fmt.Printf("\n%s  Scan with your phone camera:%s\n\n", cCyan, cReset)
	for r := 0; r < rows; r += 2 {
		fmt.Print("    ")
		for c := 0; c < cols; c++ {
			upper := r < rows && bmp[r][c]
			lower := (r+1) < rows && bmp[r+1][c]
			switch {
			case upper && lower:
				fmt.Print("█")
			case upper:
				fmt.Print("▀")
			case lower:
				fmt.Print("▄")
			default:
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
	fmt.Printf("\n%s  %s%s\n\n", cDim, url, cReset)
}

// ── Banner ────────────────────────────────────────────────────────────────────

func printBanner(ip string, port int) {
	const W = 56
	row := func(label, value string) string {
		return fmt.Sprintf("║  %-15s%-*s║", label, W-18, value)
	}

	variant := "standard"
	if gPortable {
		variant = "portable"
	}
	beepStatus := "Web Audio (synth)"
	if len(embeddedBeep) > 0 {
		beepStatus = "embedded mp3"
	}

	title := fmt.Sprintf("BarcodeHID  (%s)", variant)
	pad   := (W - len(title)) / 2

	fmt.Println()
	fmt.Println("╔" + strings.Repeat("═", W) + "╗")
	fmt.Println("║" + strings.Repeat(" ", pad) + title +
		strings.Repeat(" ", W-pad-len(title)) + "║")
	fmt.Println("╠" + strings.Repeat("═", W) + "╣")
	fmt.Println(row("Scanner",    fmt.Sprintf("https://%s:%d", ip, port)))
	fmt.Println(row("WebSocket",  fmt.Sprintf("wss://%s:%d", ip, port)))
	fmt.Println(row("HID",        gHIDMode))
	fmt.Println(row("Auto-Enter", map[bool]string{true: "ON", false: "OFF"}[gAutoEnter]))
	fmt.Println(row("Beep",       beepStatus))
	fmt.Println("╠" + strings.Repeat("─", W) + "╣")
	fmt.Println("║" + fmt.Sprintf("%-*s", W,
		"  Scan QR below with phone camera") + "║")
	fmt.Println("║" + fmt.Sprintf("%-*s", W,
		"  First visit: tap Advanced → Proceed (cert warning)") + "║")
	fmt.Println("║" + fmt.Sprintf("%-*s", W,
		"  Ctrl+C to stop") + "║")
	fmt.Println("╚" + strings.Repeat("═", W) + "╝")
}

// ── Server start ──────────────────────────────────────────────────────────────

func startServer(dir string, port int, host string) error {
	cert, err := ensureCert(dir)
	if err != nil {
		return fmt.Errorf("SSL cert: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	ip   := getLANIP()
	bind := fmt.Sprintf("%s:%d", host, port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)

	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return fmt.Errorf("listen %s: %w", bind, err)
	}

	openScanLog(dir)

	printBanner(ip, port)
	printQR(fmt.Sprintf("https://%s:%d", ip, port))
	printOK(fmt.Sprintf("Listening on https://%s:%d", ip, port))
	fmt.Println()

	go func() {
		srv := &http.Server{Handler: mux}
		if err := srv.Serve(tls.NewListener(ln, tlsCfg)); err != nil &&
			err != http.ErrServerClosed {
			printFail("Server: " + err.Error())
			os.Exit(1)
		}
	}()

	return nil
}
