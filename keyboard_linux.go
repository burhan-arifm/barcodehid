//go:build linux

package main

// keyboard_linux.go
// HID backends for Linux.
//
// Runtime priority chain:
//   1. /dev/uinput  — real kernel virtual keyboard, X11 + Wayland
//                     works without setup if user is in 'input' group
//                     setup.sh configures this permanently
//   2. dotool       — X11 + Wayland, reads stdin, no daemon needed
//   3. ydotool      — X11 + Wayland, needs ydotoold daemon
//   4. wtype        — Wayland only, no daemon
//   5. xdotool      — X11 only, no daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// ── uinput constants ──────────────────────────────────────────────────────────

const uinputPath = "/dev/uinput"

const (
	uiSetEvBit   = 0x40045564
	uiSetKeyBit  = 0x40045565
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502

	evSyn     = 0x00
	evKey     = 0x01
	synReport = 0x00

	keyShift = 42
	keyEnter = 28
)

// inputEvent mirrors Linux struct input_event.
type inputEvent struct {
	sec   uint64
	usec  uint64
	typ   uint16
	code  uint16
	value int32
}

// ── Backend 1: /dev/uinput ────────────────────────────────────────────────────

type UInputDevice struct {
	fd int
	mu sync.Mutex
}

func openUInput() (*UInputDevice, error) {
	fd, err := syscall.Open(uinputPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	ioctl := func(req, arg uintptr) error {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
		if errno != 0 {
			return errno
		}
		return nil
	}

	if err := ioctl(uiSetEvBit, evKey); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT: %w", err)
	}

	allKeys := []uint16{keyShift, keyEnter}
	for code := range keymap {
		allKeys = append(allKeys, code)
	}
	for _, code := range allKeys {
		if err := ioctl(uiSetKeyBit, uintptr(code)); err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("UI_SET_KEYBIT(%d): %w", code, err)
		}
	}

	type inputID struct{ bustype, vendor, product, version uint16 }
	type uinputUserDev struct {
		name     [80]byte
		id       inputID
		ffEffMax uint32
		absMax   [64]int32
		absMin   [64]int32
		absFuzz  [64]int32
		absFlat  [64]int32
	}
	var dev uinputUserDev
	copy(dev.name[:], "BarcodeHID Virtual Scanner")
	dev.id = inputID{bustype: 0x03, vendor: 0x4842, product: 0x4349, version: 1}

	devBytes := (*[unsafe.Sizeof(dev)]byte)(unsafe.Pointer(&dev))[:]
	if _, err := syscall.Write(fd, devBytes); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("write uinput_user_dev: %w", err)
	}
	if err := ioctl(uiDevCreate, 0); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("UI_DEV_CREATE: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return &UInputDevice{fd: fd}, nil
}

func (d *UInputDevice) emit(typ, code uint16, value int32) {
	ev := inputEvent{typ: typ, code: code, value: value}
	b  := (*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))[:]
	_, _ = syscall.Write(d.fd, b)
}

func (d *UInputDevice) syn() { d.emit(evSyn, synReport, 0) }

func (d *UInputDevice) pressKey(code uint16, shift bool) {
	if shift {
		d.emit(evKey, keyShift, 1); d.syn()
	}
	d.emit(evKey, code, 1); d.syn()
	d.emit(evKey, code, 0); d.syn()
	if shift {
		d.emit(evKey, keyShift, 0); d.syn()
	}
	time.Sleep(4 * time.Millisecond)
}

func (d *UInputDevice) TypeString(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, ch := range s {
		if m, ok := charMap[ch]; ok {
			d.pressKey(m.code, m.shift)
		} else {
			debugf("no keymap for %q — skipped", ch)
		}
	}
}

func (d *UInputDevice) PressEnter() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pressKey(keyEnter, false)
}

func (d *UInputDevice) Close() {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.fd), uiDevDestroy, 0) //nolint
	syscall.Close(d.fd)
}

func (d *UInputDevice) Mode() string {
	return "uinput (real kernel HID — X11 + Wayland)"
}

// ── Backends 2-5: command-line tools ─────────────────────────────────────────

type CmdKeyboard struct {
	modeName string
	typeFn   func(s string) error
	enterFn  func() error
	mu       sync.Mutex
}

func (c *CmdKeyboard) TypeString(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.typeFn(s); err != nil {
		debugf("%s TypeString: %v", c.modeName, err)
	}
}

func (c *CmdKeyboard) PressEnter() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.enterFn(); err != nil {
		debugf("%s PressEnter: %v", c.modeName, err)
	}
}

func (c *CmdKeyboard) Close()       {}
func (c *CmdKeyboard) Mode() string { return c.modeName }

func run(bin string, args ...string) error {
	return exec.Command(bin, args...).Run()
}

func makeDotool() (*CmdKeyboard, error) {
	path, err := exec.LookPath("dotool")
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &CmdKeyboard{
		modeName: "dotool (X11 + Wayland, no daemon)",
		typeFn: func(s string) error {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader("type " + s + "\n")
			return cmd.Run()
		},
		enterFn: func() error {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader("key enter\n")
			return cmd.Run()
		},
	}, nil
}

func makeYdotool() (*CmdKeyboard, error) {
	path, err := exec.LookPath("ydotool")
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &CmdKeyboard{
		modeName: "ydotool (X11 + Wayland, needs ydotoold)",
		typeFn:   func(s string) error { return run(path, "type", "--", s) },
		enterFn:  func() error { return run(path, "key", "enter") },
	}, nil
}

func makeWtype() (*CmdKeyboard, error) {
	path, err := exec.LookPath("wtype")
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &CmdKeyboard{
		modeName: "wtype (Wayland only)",
		typeFn:   func(s string) error { return run(path, "--", s) },
		enterFn:  func() error { return run(path, "-k", "Return") },
	}, nil
}

func makeXdotool() (*CmdKeyboard, error) {
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &CmdKeyboard{
		modeName: "xdotool (X11 only)",
		typeFn:   func(s string) error { return run(path, "type", "--clearmodifiers", "--", s) },
		enterFn:  func() error { return run(path, "key", "Return") },
	}, nil
}

// ── Selector ──────────────────────────────────────────────────────────────────

// newKeyboard tries uinput first, then walks the command-line tool chain.
// It never requires setup.sh — but uinput will be available if setup.sh
// was already run, giving real kernel HID automatically.
func newKeyboard() Keyboard {
	dev, err := openUInput()
	if err == nil {
		printOK("HID: uinput — real kernel keyboard events (X11 + Wayland)")
		return dev
	}

	// Inform user why uinput wasn't available, but don't block
	if os.IsPermission(err) {
		printInfo("uinput: no permission — run setup.sh for real HID support")
	} else {
		debugf("uinput unavailable: %v", err)
	}

	type candidate struct {
		name string
		fn   func() (*CmdKeyboard, error)
	}

	for _, c := range []candidate{
		{"dotool",  makeDotool},
		{"ydotool", makeYdotool},
		{"wtype",   makeWtype},
		{"xdotool", makeXdotool},
	} {
		kb, err := c.fn()
		if err == nil {
			printOK(fmt.Sprintf("HID: %s", kb.Mode()))
			return kb
		}
		debugf("skip %s: %v", c.name, err)
	}

	noBackendExit(
		"Run setup.sh for uinput support (best option, X11 + Wayland)",
		"Or install dotool:  https://git.sr.ht/~geb/dotool  (X11 + Wayland, no daemon)",
		"Or install wtype:   sudo apt install wtype          (Wayland only)",
		"Or install xdotool: sudo apt install xdotool        (X11 only)",
	)
	return nil
}
