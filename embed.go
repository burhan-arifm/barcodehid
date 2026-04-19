package main

import _ "embed"

//go:embed assets/scanner.html
var embeddedHTML []byte

//go:embed assets/qr.html
var embeddedQRHTML []byte

//go:embed assets/qrcode.min.js
var embeddedQRCodeJS []byte

//go:embed assets/tray-icon.png
var embeddedTrayIcon []byte
