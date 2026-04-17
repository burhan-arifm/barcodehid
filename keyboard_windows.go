//go:build windows

package main

// keyboard_windows.go
// HID backend for Windows using the SendInput Win32 API.
//
// SendInput injects keyboard events at the Win32 input queue level —
// identical to physical keyboard input from the perspective of any application.
// No external tools, no elevated privileges, no setup required.
//
// Key mapping: each rune is resolved to a Virtual Key code (VK_*) + shift
// state from the US QWERTY table in keyboard_common.go. Characters not in
// the table are sent via Unicode input events as a fallback.

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ── Win32 constants ───────────────────────────────────────────────────────────

const (
	inputKeyboard = 1

	keyeventfKeyup    = 0x0002
	keyeventfUnicode  = 0x0004
	keyeventfScancode = 0x0008

	vkShift  = 0x10
	vkReturn = 0x0D
	vkSpace  = 0x20
)

// US QWERTY: Linux keycode → Windows Virtual Key code.
// We reuse the Linux keymap codes as a common denominator.
var linuxToVK = map[uint16]uint16{
	2: '1', 3: '2', 4: '3', 5: '4', 6: '5',
	7: '6', 8: '7', 9: '8', 10: '9', 11: '0',
	12: 0xBD, // VK_OEM_MINUS  -
	13: 0xBB, // VK_OEM_PLUS   =
	16: 'Q', 17: 'W', 18: 'E', 19: 'R', 20: 'T',
	21: 'Y', 22: 'U', 23: 'I', 24: 'O', 25: 'P',
	26: 0xDB, // VK_OEM_4  [
	27: 0xDD, // VK_OEM_6  ]
	28: vkReturn,
	30: 'A', 31: 'S', 32: 'D', 33: 'F', 34: 'G',
	35: 'H', 36: 'J', 37: 'K', 38: 'L',
	39: 0xBA, // VK_OEM_1  ;
	40: 0xDE, // VK_OEM_7  '
	41: 0xC0, // VK_OEM_3  `
	43: 0xDC, // VK_OEM_5  backslash
	44: 'Z', 45: 'X', 46: 'C', 47: 'V', 48: 'B',
	49: 'N', 50: 'M',
	51: 0xBC, // VK_OEM_COMMA  ,
	52: 0xBE, // VK_OEM_PERIOD .
	53: 0xBF, // VK_OEM_2      /
	57: vkSpace,
}

// INPUT struct as defined by Win32 (KEYBDINPUT union variant).
// We define it manually to avoid CGo.
type keyboardInput struct {
	typ  uint32
	ki   keybdInput
	_pad [8]byte // union padding to match MOUSEINPUT size
}

type keybdInput struct {
	vk        uint16
	scan      uint16
	flags     uint32
	time      uint32
	extraInfo uintptr
}

// ── Win32 keyboard ────────────────────────────────────────────────────────────

var (
	user32      = windows.NewLazySystemDLL("user32.dll")
	sendInput   = user32.NewProc("SendInput")
)

type Win32Keyboard struct {
	mu sync.Mutex
}

func newWin32Keyboard() *Win32Keyboard {
	return &Win32Keyboard{}
}

func (w *Win32Keyboard) send(inputs []keyboardInput) {
	if len(inputs) == 0 {
		return
	}
	sendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}

func (w *Win32Keyboard) keyDown(vk uint16) keyboardInput {
	return keyboardInput{
		typ: inputKeyboard,
		ki:  keybdInput{vk: vk},
	}
}

func (w *Win32Keyboard) keyUp(vk uint16) keyboardInput {
	return keyboardInput{
		typ: inputKeyboard,
		ki:  keybdInput{vk: vk, flags: keyeventfKeyup},
	}
}

// unicodeDown/Up send a Unicode character directly (fallback for unmapped chars).
func (w *Win32Keyboard) unicodeDown(ch rune) keyboardInput {
	return keyboardInput{
		typ: inputKeyboard,
		ki:  keybdInput{scan: uint16(ch), flags: keyeventfUnicode},
	}
}

func (w *Win32Keyboard) unicodeUp(ch rune) keyboardInput {
	return keyboardInput{
		typ: inputKeyboard,
		ki:  keybdInput{scan: uint16(ch), flags: keyeventfUnicode | keyeventfKeyup},
	}
}

func (w *Win32Keyboard) pressVK(vk uint16, shift bool) {
	var inputs []keyboardInput
	if shift {
		inputs = append(inputs, w.keyDown(vkShift))
	}
	inputs = append(inputs, w.keyDown(vk), w.keyUp(vk))
	if shift {
		inputs = append(inputs, w.keyUp(vkShift))
	}
	w.send(inputs)
	time.Sleep(4 * time.Millisecond)
}

func (w *Win32Keyboard) pressUnicode(ch rune) {
	w.send([]keyboardInput{w.unicodeDown(ch), w.unicodeUp(ch)})
	time.Sleep(4 * time.Millisecond)
}

func (w *Win32Keyboard) TypeString(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range s {
		if ch == '\n' {
			w.pressVK(vkReturn, false)
			continue
		}
		// Look up in our charMap (Linux keycode → shift state)
		// then convert the Linux keycode to a Windows VK code.
		if m, ok := charMap[ch]; ok {
			if vk, ok2 := linuxToVK[m.code]; ok2 {
				w.pressVK(vk, m.shift)
				continue
			}
		}
		// Fallback: send as raw Unicode input event
		debugf("no VK mapping for %q — using Unicode input", ch)
		w.pressUnicode(ch)
	}
}

func (w *Win32Keyboard) PressEnter() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pressVK(vkReturn, false)
}

func (w *Win32Keyboard) Close() {}

func (w *Win32Keyboard) Mode() string {
	return "SendInput Win32 API (real HID — works in all Windows apps)"
}

// ── Selector ──────────────────────────────────────────────────────────────────

// newKeyboard returns the Win32 keyboard backend.
// portable parameter is accepted for API compatibility but has no effect on
// Windows — SendInput requires no setup on any variant.
func newKeyboard() Keyboard {
	kb := newWin32Keyboard()
	// Smoke-test: verify user32.dll + SendInput loaded correctly
	if err := sendInput.Find(); err != nil {
		printFail(fmt.Sprintf("SendInput not available: %v", err))
		printFail("This should not happen on any supported Windows version.")
		noBackendExit()
	}
	printOK("HID: SendInput Win32 API (no setup required)")
	return kb
}
