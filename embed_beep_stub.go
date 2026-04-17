//go:build !beep

package main

// No beep.mp3 at build time — phone will use Web Audio API synthesis.
var embeddedBeep []byte
