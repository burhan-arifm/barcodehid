//go:build darwin

package main

// keyboard_darwin.go
// HID backend for macOS using Core Graphics CGEvent API via CGo.
//
// NOTE: macOS tray/background mode is not yet supported.
// The app runs in foreground mode only on macOS.
// Full tray support requires proper NSApplication initialization
// which is planned for a future release.

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices

#include <CoreGraphics/CoreGraphics.h>
#include <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

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

int check_accessibility() {
    return AXIsProcessTrusted() ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// ── macOS CGKey table (US QWERTY) ─────────────────────────────────────────────

var linuxToCGKey = map[uint16]C.CGKeyCode{
	2: 18, 3: 19, 4: 20, 5: 21, 6: 23, 7: 22, 8: 26, 9: 28, 10: 25, 11: 29,
	12: 27, 13: 24,
	16: 12, 17: 13, 18: 14, 19: 15, 20: 17, 21: 16, 22: 32, 23: 34, 24: 31, 25: 35,
	26: 33, 27: 30,
	28: 36,
	30: 0, 31: 1, 32: 2, 33: 3, 34: 5, 35: 4, 36: 38, 37: 40, 38: 37,
	39: 41, 40: 39, 41: 50, 43: 42,
	44: 6, 45: 7, 46: 8, 47: 9, 48: 11, 49: 45, 50: 46,
	51: 43, 52: 47, 53: 44,
	57: 49,
}

const cgKeyReturn C.CGKeyCode = 36

// ── CGEvent keyboard ──────────────────────────────────────────────────────────

type CGKeyboard struct{ mu sync.Mutex }

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
	return "CGEvent Core Graphics"
}

// ── Selector ──────────────────────────────────────────────────────────────────

func newKeyboard() Keyboard {
	if C.check_accessibility() == 0 {
		fmt.Println()
		fmt.Println("  ┌─────────────────────────────────────────────────┐")
		fmt.Println("  │  Accessibility permission required               │")
		fmt.Println("  │                                                  │")
		fmt.Println("  │  1. System Settings → Privacy & Security        │")
		fmt.Println("  │     → Accessibility                             │")
		fmt.Println("  │  2. Click + and add BarcodeHID                  │")
		fmt.Println("  │  3. Make sure the toggle is ON                  │")
		fmt.Println("  │  4. Re-run barcodehid                           │")
		fmt.Println("  └─────────────────────────────────────────────────┘")
		fmt.Println()
		os.Exit(1)
	}
	printOK("HID: CGEvent Core Graphics")
	return &CGKeyboard{}
}
