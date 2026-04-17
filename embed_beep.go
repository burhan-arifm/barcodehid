//go:build beep

package main

import _ "embed"

//go:embed assets/beep.mp3
var embeddedBeep []byte
