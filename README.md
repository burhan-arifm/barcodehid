# BarcodeHID

> Scan barcodes with your Android phone camera → text appears on your PC as real keyboard input.

No drivers. No USB dongle. Single-file distribution. Works on Linux, Windows, and macOS.

```
Android phone (ZXing + WebSocket) ──wss──▶ barcodehid ──▶ OS keyboard events
```

---

## Downloads

Grab the latest from [Releases](../../releases/latest).

| File | Platform |
|---|---|
| `barcodehid-linux-x86_64.AppImage` | Linux (any distro, x64) |
| `barcodehid-windows-amd64.exe` | Windows x64 |
| `barcodehid-macos-amd64` | macOS Intel |
| `barcodehid-macos-arm64` | macOS Apple Silicon |

---

## How it works

1. Run `barcodehid` — a QR code appears in the terminal
2. Scan the QR with your phone camera — browser opens automatically
3. Allow camera access → scanner connects → start scanning
4. Each barcode is typed at whatever is focused on your PC, exactly like a real USB scanner

---

## Platform setup

### Linux

The AppImage runs immediately on any distro — no installation needed.

```bash
chmod +x barcodehid-linux-x86_64.AppImage
./barcodehid-linux-x86_64.AppImage
```

**Optional: upgrade to real kernel HID** (recommended for best compatibility)

By default the AppImage uses `dotool`/`wtype`/`xdotool` for input. Running
`setup.sh` once gives you real `/dev/uinput` kernel-level HID — works in
terminals, games, and any app that reads raw keycodes.

```bash
bash setup.sh      # needs sudo — sets up uinput permissions
# log out and back in
./barcodehid-linux-x86_64.AppImage
```

The app detects uinput automatically at startup. The phone UI shows
`⌨ REAL HID` (green) when it's active.

**Fallback chain** (used if uinput isn't available):
`dotool` → `ydotool` → `wtype` → `xdotool`

Install whichever matches your display server:
```bash
# dotool — best option (X11 + Wayland, no daemon)
# Build from source: https://git.sr.ht/~geb/dotool
go install git.sr.ht/~geb/dotool/cmd/dotool@latest

# wtype — Wayland only
sudo apt install wtype

# xdotool — X11 only
sudo apt install xdotool
```

### Windows

No setup needed. Just run:
```
barcodehid-windows-amd64.exe
```

If Windows Defender shows a warning: **More info → Run anyway**.
The project is open source — build it yourself to verify.

### macOS

BarcodeHID runs as a **menu bar app** — no window, no Dock icon. After
launching, look for the icon in the top-right menu bar.

**Full step-by-step guide: [docs/macos-setup.md](docs/macos-setup.md)**

Quick version:

**1. Remove the quarantine flag** (one-time — macOS blocks unsigned downloads)
```bash
xattr -d com.apple.quarantine /Applications/BarcodeHID.app
```

**2. Grant Accessibility permission** (one-time — required to simulate keyboard input)
```
System Settings → Privacy & Security → Accessibility → + → BarcodeHID → toggle ON
```

**3. Open the app**
```bash
open /Applications/BarcodeHID.app
```
A notification appears and the BarcodeHID icon shows in the menu bar.
Right-click it → **Show QR Code** to pair your phone.

> **Seeing nothing after step 1?** That means Accessibility permission is
> missing — go to step 2. The app is designed to be invisible (menu bar only)
> so a silent launch is normal until Accessibility is granted.

---

## Usage

```bash
./barcodehid                  # start (default port 8765)
./barcodehid --no-enter       # don't press Enter after each scan
./barcodehid --port 9000      # custom port
./barcodehid --host 0.0.0.0   # bind address (default: all interfaces)
./barcodehid --debug          # verbose logging
```

### Phone UI

| Element | Description |
|---|---|
| **QR code** (terminal) | Scan to open scanner — auto-connects, no typing |
| **Enter toggle** | ON = press Enter after scan (like a real scanner), OFF = value only |
| **Beep toggle** | Enable/disable scan confirmation sound |
| **⌨ REAL HID** badge | Green = uinput active, Orange = tool fallback |
| **Scan history** | Last 50 scans with timestamp and mode |

---

## Beep sound

Default: synthesized 1800 Hz tone via Web Audio API (no file needed).

Custom: place `assets/beep.mp3` before building — it gets embedded in the binary.

Generate one with ffmpeg:
```bash
ffmpeg -f lavfi -i "sine=frequency=1800:duration=0.08" \
  -codec:a libmp3lame -qscale:a 9 assets/beep.mp3
```

---

## Supported barcode formats

1D: EAN-13, EAN-8, UPC-A, UPC-E, Code 128, Code 39, Code 93, ITF, Codabar

QR code support: planned.

---

## Build from source

### Quick (using build.sh)

```bash
git clone https://github.com/yourusername/barcodehid
cd barcodehid

# Optional: embed a custom beep
cp /path/to/beep.mp3 assets/beep.mp3

# Build binary (+ AppImage on Linux)
bash build.sh

# Linux only: setup uinput for real HID (optional but recommended)
bash setup.sh
```

`build.sh` installs Go automatically if not found, and downloads
`appimagetool` automatically on Linux.

### Manual

#### 1. Install Go (1.21+)

```bash
# Debian / Ubuntu
sudo apt-get install golang-go

# Fedora
sudo dnf install golang

# Arch
sudo pacman -S go

# macOS
brew install go

# Windows
winget install GoLang.Go

# Any platform — official binary
# https://go.dev/dl/
```

Verify: `go version`

#### 2. Get the code

```bash
git clone https://github.com/yourusername/barcodehid
cd barcodehid
go mod download
```

#### 3. Build

**Linux / macOS:**
```bash
# Without beep
go build -ldflags="-s -w -X main.buildVariant=portable" -trimpath -o barcodehid .

# With embedded beep (place assets/beep.mp3 first)
go build -tags beep -ldflags="-s -w -X main.buildVariant=portable" -trimpath -o barcodehid .
```

**Windows** (from Linux, cross-compile):
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -trimpath -o barcodehid.exe .
```

**macOS** (must build on macOS — requires CGo for Core Graphics):
```bash
go build -ldflags="-s -w -X main.buildVariant=portable" -trimpath -o barcodehid .
```

#### 4. Build AppImage (Linux only)

```bash
# Download appimagetool
curl -fsSL -o appimagetool \
  "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
chmod +x appimagetool

# Build AppDir
mkdir -p AppDir/usr/bin AppDir/usr/share/icons/hicolor/scalable/apps
cp barcodehid                                      AppDir/usr/bin/barcodehid
cp packaging/appimage/AppRun                       AppDir/AppRun
cp packaging/appimage/barcodehid.desktop           AppDir/barcodehid.desktop
cp packaging/appimage/barcodehid.svg               AppDir/barcodehid.svg
cp packaging/appimage/barcodehid.svg               AppDir/usr/share/icons/hicolor/scalable/apps/barcodehid.svg
chmod +x AppDir/usr/bin/barcodehid AppDir/AppRun

# Package
ARCH=x86_64 ./appimagetool AppDir barcodehid-linux-x86_64.AppImage
chmod +x barcodehid-linux-x86_64.AppImage
```

---

## Troubleshooting

**Camera doesn't open on phone**
→ Open `https://` not `http://`. Camera requires a secure context.
→ Proceed past the cert warning (Advanced → Proceed) on first visit.

**Linux: nothing gets typed**
→ Install dotool, wtype, or xdotool (see Linux setup above).
→ Or run `setup.sh` for uinput real HID.

**macOS: nothing gets typed**
→ Check Accessibility: System Settings → Privacy & Security → Accessibility.

**Windows: binary blocked by Defender**
→ More info → Run anyway. Or build from source.

**Phone: "Connection failed"**
→ Phone and PC must be on the same network.
→ Check firewall: `sudo ufw allow 8765` (Linux).

**Scans appear in the wrong window**
→ Click the target input on your PC before scanning.
