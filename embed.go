package main

import _ "embed"

// scanner.html is always required at build time.
//
//go:embed assets/scanner.html
var embeddedHTML []byte

// beep.mp3 is optional. If assets/beep.mp3 exists at build time it is
// embedded here; if not, the file is empty and the phone falls back to
// Web Audio API synthesis. We use a build tag to switch between the two
// variants — see embed_beep.go and embed_beep_stub.go.
