#!/usr/bin/env bash
# =============================================================================
#  BarcodeHID — Build script
#
#  Builds a single portable binary for your current platform.
#  On Linux: also builds an AppImage if appimagetool is available.
#
#  Usage:
#    bash build.sh                  # build binary (+ AppImage on Linux)
#    bash build.sh --no-appimage    # skip AppImage even on Linux
#    bash build.sh --appimage-only  # skip binary rebuild, just repackage AppImage
# =============================================================================
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
ok()   { echo -e "${GREEN}  ✔  $1${NC}"; }
info() { echo -e "${YELLOW}  →  $1${NC}"; }
fail() { echo -e "${RED}  ✘  $1${NC}"; exit 1; }

DO_BINARY=1
DO_APPIMAGE=1
OUT_DIR="$DIR"   # default: same directory as build.sh
for arg in "$@"; do
  case "$arg" in
    --no-appimage)   DO_APPIMAGE=0 ;;
    --appimage-only) DO_BINARY=0 ;;
    --out-dir=*)     OUT_DIR="${arg#--out-dir=}" ;;
    --out-dir)       shift; OUT_DIR="$1" ;;  # handled below via next arg
  esac
done
# Handle --out-dir as separate arg (not = form)
args=("$@")
for i in "${!args[@]}"; do
  if [ "${args[$i]}" = "--out-dir" ] && [ -n "${args[$i+1]}" ]; then
    OUT_DIR="${args[$i+1]}"
  fi
done
# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"   # resolve to absolute path

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║         BarcodeHID — Build                       ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# ── Detect OS ─────────────────────────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  EXT="" ;;
  darwin) EXT="" ;;
  mingw*|msys*|cygwin*) EXT=".exe" ;;
  *) EXT="" ;;
esac

# ── Find or install Go ────────────────────────────────────────────────────────
find_go() {
  for p in go /usr/local/go/bin/go /usr/lib/go/bin/go \
            "$HOME/go/bin/go" "$HOME/.local/go/bin/go"; do
    command -v "$p" &>/dev/null && echo "$p" && return 0
  done
  return 1
}

GO=$(find_go || true)

if [ -z "$GO" ]; then
  info "Go not found — installing..."
  if   command -v apt-get &>/dev/null; then sudo apt-get update -qq && sudo apt-get install -y golang-go
  elif command -v dnf     &>/dev/null; then sudo dnf install -y golang
  elif command -v pacman  &>/dev/null; then sudo pacman -Sy --noconfirm go
  elif command -v brew    &>/dev/null; then brew install go
  else fail "Cannot auto-install Go. Download from https://go.dev/dl/ (need >= 1.21)"; fi
  GO=$(find_go || fail "Go still not found after install.")
fi

ok "Go: $GO ($("$GO" version | awk '{print $3}'))"

# ── Validate required files ────────────────────────────────────────────────────
[ -f "main.go"              ] || fail "main.go not found — run from the project root"
[ -f "assets/scanner.html"  ] || fail "assets/scanner.html not found"
[ -f "assets/qr.html"       ] || fail "assets/qr.html not found"

# ── Download qrcode.min.js if missing ─────────────────────────────────────────
# The library is embedded in the binary at build time via go:embed.
# Source: davidshimjs/qrcodejs (MIT license) via npm registry.
if [ ! -f "assets/qrcode.min.js" ]; then
  info "Downloading qrcode.min.js (davidshimjs/qrcodejs)..."
  TMP_TGZ="$(mktemp)"
  TMP_DIR="$(mktemp -d)"
  if curl -fsSL "https://registry.npmjs.org/qrcodejs2/-/qrcodejs2-0.0.2.tgz"       -o "$TMP_TGZ" 2>/dev/null; then
    tar -xzf "$TMP_TGZ" -C "$TMP_DIR" 2>/dev/null
    if [ -f "$TMP_DIR/package/qrcode.min.js" ]; then
      cp "$TMP_DIR/package/qrcode.min.js" "assets/qrcode.min.js"
      ok "qrcode.min.js downloaded and saved to assets/"
    else
      fail "qrcode.min.js not found in package. Download manually from https://davidshimjs.github.io/qrcodejs/"
    fi
  else
    fail "Could not download qrcode.min.js. Download manually and place at assets/qrcode.min.js"
  fi
  rm -rf "$TMP_TGZ" "$TMP_DIR"
else
  ok "assets/qrcode.min.js already present"
fi

# ── Install Linux CGo dependencies (required for systray) ─────────────────────
if [ "$OS" = "linux" ] && [ "$DO_BINARY" = "1" ]; then
  if ! pkg-config --exists ayatana-appindicator3-0.1 2>/dev/null; then
    info "Installing Linux build dependencies for systray..."
    if command -v apt-get &>/dev/null; then
      sudo apt-get update -qq
      sudo apt-get install -y \
        libayatana-appindicator3-dev \
        libgtk-3-dev \
        libglib2.0-dev
    elif command -v dnf &>/dev/null; then
      sudo dnf install -y \
        libayatana-appindicator-gtk3-devel \
        gtk3-devel
    elif command -v pacman &>/dev/null; then
      sudo pacman -Sy --noconfirm \
        libayatana-appindicator \
        gtk3
    else
      fail "Cannot auto-install systray deps. Install libayatana-appindicator3-dev + libgtk-3-dev manually."
    fi
    ok "Linux build dependencies installed"
  else
    ok "Linux build dependencies already present"
  fi
fi

# ── Download Go module dependencies ───────────────────────────────────────────
info "Downloading dependencies..."
"$GO" mod download
ok "Dependencies ready"

# ── Build tags ────────────────────────────────────────────────────────────────
TAGS=""
if [ -f "assets/beep.mp3" ]; then
  ok "assets/beep.mp3 found — embedding beep sound"
  TAGS="-tags beep"
else
  info "No assets/beep.mp3 — phone will use Web Audio synth beep"
  info "  Place a beep.mp3 in assets/ and rebuild to embed a custom sound"
fi

OUT="barcodehid${EXT}"

# ── Build binary ───────────────────────────────────────────────────────────────
if [ "$DO_BINARY" = "1" ]; then
  info "Building $OUT..."

  # CGo required on Linux for systray (GTK); not needed on other platforms
  CGO_FLAG="CGO_ENABLED=0"
  [ "$OS" = "linux" ] && CGO_FLAG="CGO_ENABLED=1"

  env $CGO_FLAG "$GO" build \
    $TAGS \
    -ldflags="-s -w -X main.buildVariant=portable" \
    -trimpath \
    -o "$OUT_DIR/$OUT" \
    .

  chmod +x "$OUT_DIR/$OUT" 2>/dev/null || true
  ok "Binary: $OUT_DIR/$OUT ($(du -sh "$OUT_DIR/$OUT" | cut -f1))"
fi

# ── AppImage (Linux only) ─────────────────────────────────────────────────────
if [ "$OS" = "linux" ] && [ "$DO_APPIMAGE" = "1" ]; then
  info "Building AppImage..."

  # Validate AppImage assets
  [ -f "packaging/appimage/AppRun"            ] || fail "packaging/appimage/AppRun not found"
  [ -f "packaging/appimage/barcodehid.desktop" ] || fail "packaging/appimage/barcodehid.desktop not found"
  [ -f "packaging/appimage/barcodehid.png"    ] || fail "packaging/appimage/barcodehid.png not found"
  [ -f "packaging/appimage/barcodehid.svg"    ] || fail "packaging/appimage/barcodehid.svg not found"

  # Find or download appimagetool
  # Check system PATH first, then our cache location
  APPIMAGETOOL=$(command -v appimagetool 2>/dev/null || true)
  if [ -z "$APPIMAGETOOL" ] && [ -x "$HOME/.cache/barcodehid/appimagetool" ]; then
    APPIMAGETOOL="$HOME/.cache/barcodehid/appimagetool"
    ok "appimagetool found in cache"
  fi

  if [ -z "$APPIMAGETOOL" ]; then
    info "appimagetool not found — downloading..."
    TOOL_PATH="$DIR/.appimagetool"
    curl -fsSL -o "$TOOL_PATH" \
      "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "$TOOL_PATH"

    # appimagetool is itself an AppImage — needs FUSE or extract mode
    if ! fusermount -V &>/dev/null 2>&1; then
      info "FUSE not available — using --appimage-extract-and-run mode"
      APPIMAGETOOL_ARGS="--appimage-extract-and-run"
    fi
    APPIMAGETOOL="$TOOL_PATH"
    ok "appimagetool downloaded"
  fi

  # Build AppDir
  APPDIR="$(mktemp -d)"
  trap 'rm -rf "$APPDIR"' EXIT

  mkdir -p "$APPDIR/usr/bin"
  mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"
  mkdir -p "$APPDIR/usr/share/icons/hicolor/scalable/apps"

  cp "$OUT_DIR/$OUT"                                    "$APPDIR/usr/bin/barcodehid"
  cp "$DIR/packaging/appimage/AppRun"                  "$APPDIR/AppRun"
  cp "$DIR/packaging/appimage/barcodehid.desktop"      "$APPDIR/barcodehid.desktop"
  # PNG at root: used by AppImage runtime + KDE for launcher icon
  cp "$DIR/packaging/appimage/barcodehid.png"          "$APPDIR/barcodehid.png"
  # Icons in hicolor tree: used by desktop environments
  cp "$DIR/packaging/appimage/barcodehid.png"          "$APPDIR/usr/share/icons/hicolor/256x256/apps/barcodehid.png"
  cp "$DIR/packaging/appimage/barcodehid.svg"          "$APPDIR/usr/share/icons/hicolor/scalable/apps/barcodehid.svg"

  chmod +x "$APPDIR/usr/bin/barcodehid" "$APPDIR/AppRun"

  APPIMAGE_OUT="$OUT_DIR/barcodehid-linux-x86_64.AppImage"
  ARCH=x86_64 "$APPIMAGETOOL" ${APPIMAGETOOL_ARGS:-} "$APPDIR" "$APPIMAGE_OUT" 2>/dev/null
  chmod +x "$APPIMAGE_OUT"

  ok "AppImage: barcodehid-linux-x86_64.AppImage ($(du -sh "$APPIMAGE_OUT" | cut -f1))"
fi

# ── uinput permissions (optional, for real HID) ───────────────────────────────
# Run setup.sh separately if you want /dev/uinput real HID support.
# The app works without it via dotool/wtype/xdotool fallback.
# sudo bash setup.sh

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║  ✅  Build complete!                             ║"
echo "╠══════════════════════════════════════════════════╣"
[ "$DO_BINARY" = "1" ] && \
  echo "║  Binary:   ./$OUT$(printf '%*s' $((38-${#OUT})) '')║"
[ "$OS" = "linux" ] && [ "$DO_APPIMAGE" = "1" ] && \
  echo "║  AppImage: ./barcodehid-linux-x86_64.AppImage    ║"
echo "║                                                  ║"
echo "║  Run modes:                                      ║"
echo "║    ./barcodehid              foreground (terminal)║"
echo "║    ./barcodehid --tray       background (tray)   ║"
echo "║    ./barcodehid.AppImage     background (tray)   ║"
echo "║    ./barcodehid.AppImage --foreground  terminal  ║"
echo "║                                                  ║"
echo "║  Other flags: --no-enter  --port N  --debug      ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""
