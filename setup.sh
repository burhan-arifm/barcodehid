#!/usr/bin/env bash
# =============================================================================
#  BarcodeHID — uinput permission setup (optional)
#
#  Configures /dev/uinput access so BarcodeHID can use real kernel-level
#  HID keyboard events (works on both X11 and Wayland).
#
#  Without this, the app uses dotool/wtype/xdotool as fallback (GUI apps only).
#  Run once after install. Requires sudo. Safe to re-run.
#
#  Usage:
#    bash setup.sh
# =============================================================================
set -e

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}  ✔  $1${NC}"; }
info() { echo -e "${YELLOW}  →  $1${NC}"; }

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║     BarcodeHID — uinput permission setup         ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# ── uinput kernel module ──────────────────────────────────────────────────────
info "Loading uinput kernel module..."
sudo modprobe uinput
grep -q "^uinput" /etc/modules-load.d/uinput.conf 2>/dev/null \
  || echo "uinput" | sudo tee /etc/modules-load.d/uinput.conf > /dev/null
ok "uinput module loaded and set to load on boot"

# ── udev rule ─────────────────────────────────────────────────────────────────
RULE='SUBSYSTEM=="misc", KERNEL=="uinput", MODE="0660", GROUP="input"'
FILE="/etc/udev/rules.d/99-barcodehid.rules"
if [ ! -f "$FILE" ] || ! grep -qF "$RULE" "$FILE"; then
  info "Creating udev rule..."
  echo "$RULE" | sudo tee "$FILE" > /dev/null
  sudo udevadm control --reload-rules && sudo udevadm trigger
  ok "udev rule created: $FILE"
else
  ok "udev rule already exists"
fi

# ── input group ───────────────────────────────────────────────────────────────
NEED_RELOGIN=0
if ! groups "$USER" | grep -q '\binput\b'; then
  info "Adding $USER to 'input' group..."
  sudo usermod -aG input "$USER"
  NEED_RELOGIN=1
  ok "Added — log out and back in for this to take effect"
else
  ok "$USER already in 'input' group"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║  ✅  Setup complete!                             ║"
echo "╠══════════════════════════════════════════════════╣"
if [ "$NEED_RELOGIN" = "1" ]; then
echo "║  ⚠  Log out and back in, then run BarcodeHID.   ║"
else
echo "║  Real HID via uinput is now active.              ║"
echo "║  Run BarcodeHID — tray will show ⌨ REAL HID.   ║"
fi
echo "╚══════════════════════════════════════════════════╝"
echo ""
