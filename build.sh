#!/usr/bin/env bash
# =============================================================================
#  BarcodeHID — Build script
#
#  Builds a single portable binary for your current platform.
#  On Linux: also builds an AppImage if appimagetool is available.
#
#  Usage:
#    bash build.sh                # build binary (+ AppImage on Linux)
#    bash build.sh --no-appimage  # skip AppImage even on Linux
#    bash build.sh --appimage-only  # skip binary rebuild, just repackage
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
for arg in "$@"; do
  case "$arg" in
    --no-appimage)   DO_APPIMAGE=0 ;;
    --appimage-only) DO_BINARY=0 ;;
  esac
done

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

# ── Validate ──────────────────────────────────────────────────────────────────
[ -f "main.go"             ] || fail "main.go not found — run from the project root"
[ -f "assets/scanner.html" ] || fail "assets/scanner.html not found"

# ── Dependencies ──────────────────────────────────────────────────────────────
info "Downloading dependencies..."
"$GO" mod download
ok "Dependencies ready"

# ── Build tags ────────────────────────────────────────────────────────────────
TAGS=""
if [ -f "assets/beep.mp3" ]; then
  ok "assets/beep.mp3 found — will embed beep sound"
  TAGS="-tags beep"
else
  info "No assets/beep.mp3 — phone will use Web Audio synth beep"
fi

OUT="barcodehid${EXT}"

# ── Build binary ──────────────────────────────────────────────────────────────
if [ "$DO_BINARY" = "1" ]; then
  info "Building $OUT..."

  "$GO" build \
    $TAGS \
    -ldflags="-s -w -X main.buildVariant=portable" \
    -trimpath \
    -o "$OUT" \
    .

  chmod +x "$OUT" 2>/dev/null || true
  ok "Binary: $OUT ($(du -sh "$OUT" | cut -f1))"
fi

# ── AppImage (Linux only) ─────────────────────────────────────────────────────
if [ "$OS" = "linux" ] && [ "$DO_APPIMAGE" = "1" ]; then
  info "Building AppImage..."

  # Find or download appimagetool
  APPIMAGETOOL=$(command -v appimagetool 2>/dev/null || true)

  if [ -z "$APPIMAGETOOL" ]; then
    info "appimagetool not found — downloading..."
    TOOL_PATH="$DIR/.appimagetool"
    curl -fsSL -o "$TOOL_PATH" \
      "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "$TOOL_PATH"
    APPIMAGETOOL="$TOOL_PATH"
    ok "appimagetool downloaded"
  fi

  # Build AppDir structure
  APPDIR="$(mktemp -d)"
  trap 'rm -rf "$APPDIR"' EXIT

  mkdir -p "$APPDIR/usr/bin"
  mkdir -p "$APPDIR/usr/share/icons/hicolor/scalable/apps"

  cp "$DIR/$OUT"                                    "$APPDIR/usr/bin/barcodehid"
  cp "$DIR/packaging/appimage/AppRun"               "$APPDIR/AppRun"
  cp "$DIR/packaging/appimage/barcodehid.desktop"   "$APPDIR/barcodehid.desktop"
  cp "$DIR/packaging/appimage/barcodehid.svg"       "$APPDIR/barcodehid.svg"
  cp "$DIR/packaging/appimage/barcodehid.svg"       "$APPDIR/usr/share/icons/hicolor/scalable/apps/barcodehid.svg"
  chmod +x "$APPDIR/usr/bin/barcodehid" "$APPDIR/AppRun"

  APPIMAGE_OUT="$DIR/barcodehid-linux-x86_64.AppImage"
  ARCH=x86_64 "$APPIMAGETOOL" "$APPDIR" "$APPIMAGE_OUT" 2>/dev/null
  chmod +x "$APPIMAGE_OUT"

  ok "AppImage: barcodehid-linux-x86_64.AppImage ($(du -sh "$APPIMAGE_OUT" | cut -f1))"
fi

# ── Linux: configure uinput (optional, improves HID quality) ─────────────────
if [ "$OS" = "linux" ] && [ -f "setup.sh" ] && [ "$DO_BINARY" = "1" ]; then
  info "Configuring uinput permissions (optional, improves HID quality)..."
  bash setup.sh --quiet
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║  ✅  Build complete!                             ║"
echo "╠══════════════════════════════════════════════════╣"
[ "$DO_BINARY" = "1" ]    && echo "║  Binary:   ./$OUT$(printf '%*s' $((38-${#OUT})) '')║"
[ "$OS" = "linux" ] && [ "$DO_APPIMAGE" = "1" ] && \
  echo "║  AppImage: ./barcodehid-linux-x86_64.AppImage    ║"
echo "║                                                  ║"
echo "║  Flags: --no-enter  --port N  --debug            ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""
