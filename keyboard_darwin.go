//go:build darwin

package main

// keyboard_darwin.go
// HID backend for macOS using Core Graphics CGEvent API via CGo.
//
// CGEventCreateKeyboardEvent posts keyboard events directly into the Quartz
// event stream — the same level as physical keyboard input. Works in all
// macOS applications including Terminal, browsers, and native apps.
//
// IMPORTANT: macOS requires Accessibility permission before this works.
// System Settings → Privacy & Security → Accessibility → add barcodehid
// The binary will detect missing permission and print a clear guided message.

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices

#include <CoreGraphics/CoreGraphics.h>
#include <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

// post_key sends a single key event (down or up) with optional shift modifier.
void post_key(CGKeyCode keyCode, bool keyDown, bool shift) {
    CGEventFlags flags = 0;
    if (shift) flags |= kCGEventFlagMaskShift;

    CGEventRef event = CGEventCreateKeyboardEvent(NULL, keyCode, keyDown);
    if (event) {
        CGEventSetFlags(event, flags);
        CGEventPost(kCGHIDEventTap, event);
        CFRelease(event);
    }
}

// check_accessibility returns 1 if the process has Accessibility permission.
// AXIsProcessTrusted() is declared in ApplicationServices/ApplicationServices.h.
int check_accessibility() {
    return AXIsProcessTrusted() ? 1 : 0;
}
*/
import "C"

import (
	"sync"
	"time"
)

// ── macOS CGKey table (US QWERTY) ─────────────────────────────────────────────
// Maps Linux keycode (from keyboard_common.go keymap) → macOS CGKeyCode.
// Keycodes from HIToolbox/Events.h (kVK_* constants).

var linuxToCGKey = map[uint16]C.CGKeyCode{
	2:  18, // kVK_ANSI_1
	3:  19, // kVK_ANSI_2
	4:  20, // kVK_ANSI_3
	5:  21, // kVK_ANSI_4
	6:  23, // kVK_ANSI_5
	7:  22, // kVK_ANSI_6
	8:  26, // kVK_ANSI_7
	9:  28, // kVK_ANSI_8
	10: 25, // kVK_ANSI_9
	11: 29, // kVK_ANSI_0
	12: 27, // kVK_ANSI_Minus
	13: 24, // kVK_ANSI_Equal
	16: 12, // kVK_ANSI_Q
	17: 13, // kVK_ANSI_W
	18: 14, // kVK_ANSI_E
	19: 15, // kVK_ANSI_R
	20: 17, // kVK_ANSI_T
	21: 16, // kVK_ANSI_Y
	22: 32, // kVK_ANSI_U
	23: 34, // kVK_ANSI_I
	24: 31, // kVK_ANSI_O
	25: 35, // kVK_ANSI_P
	26: 33, // kVK_ANSI_LeftBracket
	27: 30, // kVK_ANSI_RightBracket
	28: 36, // kVK_Return
	30: 0,  // kVK_ANSI_A
	31: 1,  // kVK_ANSI_S
	32: 2,  // kVK_ANSI_D
	33: 3,  // kVK_ANSI_F
	34: 5,  // kVK_ANSI_G
	35: 4,  // kVK_ANSI_H
	36: 38, // kVK_ANSI_J
	37: 40, // kVK_ANSI_K
	38: 37, // kVK_ANSI_L
	39: 41, // kVK_ANSI_Semicolon
	40: 39, // kVK_ANSI_Quote
	41: 50, // kVK_ANSI_Grave
	43: 42, // kVK_ANSI_Backslash
	44: 6,  // kVK_ANSI_Z
	45: 7,  // kVK_ANSI_X
	46: 8,  // kVK_ANSI_C
	47: 9,  // kVK_ANSI_V
	48: 11, // kVK_ANSI_B
	49: 45, // kVK_ANSI_N
	50: 46, // kVK_ANSI_M
	51: 43, // kVK_ANSI_Comma
	52: 47, // kVK_ANSI_Period
	53: 44, // kVK_ANSI_Slash
	57: 49, // kVK_Space
}

const cgKeyReturn C.CGKeyCode = 36

// ── CGEvent keyboard ──────────────────────────────────────────────────────────

type CGKeyboard struct {
	mu sync.Mutex
}

func (k *CGKeyboard) pressKey(cgCode C.CGKeyCode, shift bool) {
	C.post_key(cgCode, C.bool(true), C.bool(shift))
	time.Sleep(2 * time.Millisecond)
	C.post_key(cgCode, C.bool(false), C.bool(shift))
	time.Sleep(4 * time.Millisecond)
}

func (k *CGKeyboard) TypeString(s string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, ch := range s {
		if ch == '\n' {
			k.pressKey(cgKeyReturn, false)
			continue
		}
		if m, ok := charMap[ch]; ok {
			if cgCode, ok2 := linuxToCGKey[m.code]; ok2 {
				k.pressKey(cgCode, m.shift)
				continue
			}
		}
		debugf("no CGKey mapping for %q — skipped", ch)
	}
}

func (k *CGKeyboard) PressEnter() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.pressKey(cgKeyReturn, false)
}

func (k *CGKeyboard) Close() {}

func (k *CGKeyboard) Mode() string {
	return "CGEvent Core Graphics (real HID — X11 + Wayland equivalent on macOS)"
}

// ── Accessibility check ───────────────────────────────────────────────────────

func checkAccessibility() bool {
	return C.check_accessibility() == 1
}

// ── Selector ──────────────────────────────────────────────────────────────────

// newKeyboard returns the CGEvent keyboard backend.
// portable parameter accepted for API compatibility — no effect on macOS since
// both variants use the same CGEvent API.
func newKeyboard() Keyboard {
	if !checkAccessibility() {
		// Don't exit — return a stub keyboard that prompts for permission
		// on first use. This allows the tray to start normally so the user
		// can see the notification and menu bar icon.
		printWarn("Accessibility permission not granted — keyboard simulation disabled")
		printWarn("Grant permission: System Settings → Privacy & Security → Accessibility")
		return &NoPermKeyboard{}
	}

	printOK("HID: CGEvent Core Graphics API (X11 + Wayland equivalent on macOS)")
	return &CGKeyboard{}
}

// NoPermKeyboard is a stub used when Accessibility permission is missing.
// It shows a notification on first scan attempt prompting the user to
// grant permission, rather than crashing the app on startup.
type NoPermKeyboard struct{ notified bool }

func (n *NoPermKeyboard) TypeString(s string) {
	if !n.notified {
		n.notified = true
		notify(
			"Accessibility permission required",
			"BarcodeHID cannot type keyboard input.\n\n"+
				"System Settings → Privacy & Security → Accessibility\n"+
				"→ add BarcodeHID → toggle ON → restart app",
		)
	}
}

func (n *NoPermKeyboard) PressEnter() {}
func (n *NoPermKeyboard) Close()      {}
func (n *NoPermKeyboard) Mode() string {
	return "no permission (Accessibility access required)"
}
