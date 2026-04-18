#!/usr/bin/env bash
# =============================================================================
#  BarcodeHID — macOS .app bundle maker
#
#  Wraps the compiled binary into a proper macOS .app bundle with icon.
#  Run after go build, before distribution.
#
#  Usage:
#    bash packaging/macos/make-app-bundle.sh <binary> <output-dir>
#
#  Example:
#    bash packaging/macos/make-app-bundle.sh barcodehid-macos-arm64 dist/
#    → produces dist/BarcodeHID.app and dist/BarcodeHID-arm64.app.zip
# =============================================================================
set -e

BINARY="${1:?Usage: $0 <binary> <output-dir>}"
OUTDIR="${2:?Usage: $0 <binary> <output-dir>}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}  ✔  $1${NC}"; }
info() { echo -e "${YELLOW}  →  $1${NC}"; }

# Detect arch from binary name for zip filename
ARCH="universal"
[[ "$BINARY" == *"arm64"* ]] && ARCH="arm64"
[[ "$BINARY" == *"amd64"* ]] && ARCH="amd64"

APP_NAME="BarcodeHID"
APP_DIR="$OUTDIR/${APP_NAME}.app"
CONTENTS="$APP_DIR/Contents"

info "Building ${APP_NAME}.app from $BINARY..."

# ── Bundle structure ──────────────────────────────────────────────────────────
rm -rf "$APP_DIR"
mkdir -p "$CONTENTS/MacOS"
mkdir -p "$CONTENTS/Resources"

# ── Binary ────────────────────────────────────────────────────────────────────
cp "$BINARY" "$CONTENTS/MacOS/barcodehid"
chmod +x "$CONTENTS/MacOS/barcodehid"

# ── Icon ──────────────────────────────────────────────────────────────────────
ICNS="$SCRIPT_DIR/barcodehid.icns"
if [ -f "$ICNS" ]; then
  cp "$ICNS" "$CONTENTS/Resources/${APP_NAME}.icns"
  ok "Icon embedded"
else
  echo "  ⚠  barcodehid.icns not found — bundle will have no icon"
fi

# ── Info.plist ────────────────────────────────────────────────────────────────
# LSUIElement=true: app doesn't appear in Dock (menu bar / tray app)
# NSHighResolutionCapable: Retina support
cat > "$CONTENTS/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>BarcodeHID</string>

  <key>CFBundleDisplayName</key>
  <string>BarcodeHID</string>

  <key>CFBundleIdentifier</key>
  <string>io.barcodehid</string>

  <key>CFBundleVersion</key>
  <string>1.0.0</string>

  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>

  <key>CFBundleExecutable</key>
  <string>barcodehid</string>

  <key>CFBundleIconFile</key>
  <string>BarcodeHID</string>

  <key>CFBundlePackageType</key>
  <string>APPL</string>

  <key>CFBundleSignature</key>
  <string>????</string>

  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>

  <key>LSUIElement</key>
  <true/>

  <key>NSHighResolutionCapable</key>
  <true/>

  <key>NSHumanReadableCopyright</key>
  <string>MIT License</string>

  <key>NSAccessibilityUsageDescription</key>
  <string>BarcodeHID needs Accessibility access to simulate keyboard input from barcode scans.</string>
</dict>
</plist>
PLIST

ok "Info.plist written (LSUIElement=true — no Dock icon)"

# ── Zip for distribution ──────────────────────────────────────────────────────
ZIP_NAME="${APP_NAME}-macos-${ARCH}.app.zip"
ZIP_PATH="$OUTDIR/$ZIP_NAME"

cd "$OUTDIR"
zip -r -q "$ZIP_NAME" "${APP_NAME}.app"
cd - > /dev/null

ok "Zipped: $ZIP_PATH ($(du -sh "$ZIP_PATH" | cut -f1))"

# Clean up unzipped .app (CI only needs the zip)
rm -rf "$APP_DIR"

echo ""
ok "Done → $ZIP_PATH"
