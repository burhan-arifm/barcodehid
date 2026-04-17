package main

// keyboard_common.go
// Shared Keyboard interface and US QWERTY key table used by all platforms.

import (
	"fmt"
	"os"
)

// ── Keyboard interface ────────────────────────────────────────────────────────

// Keyboard is the single interface all platform backends implement.
// TypeString sends each character as real keydown+keyup events.
// PressEnter sends a Return/Enter key event.
// Close releases any OS resources (file descriptors, handles, etc.).
// Mode returns a human-readable description of the active backend.
type Keyboard interface {
	TypeString(s string)
	PressEnter()
	Close()
	Mode() string
}

// ── US QWERTY key table ───────────────────────────────────────────────────────
// Maps Linux input keycodes → (unshifted rune, shifted rune).
// Used directly by the Linux uinput backend.
// Windows and macOS backends use the rune→VK/CGKey lookup built from this.

var keymap = map[uint16][2]rune{
	2:  {'1', '!'}, 3:  {'2', '@'}, 4:  {'3', '#'}, 5:  {'4', '$'},
	6:  {'5', '%'}, 7:  {'6', '^'}, 8:  {'7', '&'}, 9:  {'8', '*'},
	10: {'9', '('}, 11: {'0', ')'}, 12: {'-', '_'}, 13: {'=', '+'},
	16: {'q', 'Q'}, 17: {'w', 'W'}, 18: {'e', 'E'}, 19: {'r', 'R'},
	20: {'t', 'T'}, 21: {'y', 'Y'}, 22: {'u', 'U'}, 23: {'i', 'I'},
	24: {'o', 'O'}, 25: {'p', 'P'}, 26: {'[', '{'}, 27: {']', '}'},
	28: {'\n', '\n'},
	30: {'a', 'A'}, 31: {'s', 'S'}, 32: {'d', 'D'}, 33: {'f', 'F'},
	34: {'g', 'G'}, 35: {'h', 'H'}, 36: {'j', 'J'}, 37: {'k', 'K'},
	38: {'l', 'L'}, 39: {';', ':'}, 40: {'\'', '"'}, 41: {'`', '~'},
	43: {'\\', '|'},
	44: {'z', 'Z'}, 45: {'x', 'X'}, 46: {'c', 'C'}, 47: {'v', 'V'},
	48: {'b', 'B'}, 49: {'n', 'N'}, 50: {'m', 'M'}, 51: {',', '<'},
	52: {'.', '>'}, 53: {'/', '?'},
	57: {' ', ' '},
}

// charKey is a reverse-lookup entry: which keycode + shift state produces a rune.
type charKey struct {
	code  uint16
	shift bool
}

// charMap is built once at init from keymap.
var charMap map[rune]charKey

func init() {
	charMap = make(map[rune]charKey, len(keymap)*2)
	for code, chars := range keymap {
		if _, ok := charMap[chars[0]]; !ok {
			charMap[chars[0]] = charKey{code, false}
		}
		if _, ok := charMap[chars[1]]; !ok {
			charMap[chars[1]] = charKey{code, true}
		}
	}
}

// ── Helpers used by platform selectors ───────────────────────────────────────

func noBackendExit(hints ...string) {
	printFail("No HID keyboard backend is available.")
	for _, h := range hints {
		printFail("  " + h)
	}
	os.Exit(1)
}

func warnFallback(from, to string, err error) {
	printWarn(fmt.Sprintf("%s unavailable (%v) — trying %s", from, err, to))
}
