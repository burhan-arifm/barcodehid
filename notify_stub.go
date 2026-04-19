//go:build !linux

package main

// notify_stub.go
// Desktop notifications for Windows and macOS.
//
// macOS:   osascript (always available, no dependencies)
// Windows: PowerShell toast (Windows 10+, no dependencies)

import (
	"fmt"
	"os/exec"
	"runtime"
)

func notify(title, body string) {
	switch runtime.GOOS {

	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		_ = exec.Command("osascript", "-e", script).Run()

	case "windows":
		script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType=WindowsRuntime] | Out-Null
$template = @"
<toast>
  <visual>
    <binding template="ToastGeneric">
      <text>%s</text>
      <text>%s</text>
    </binding>
  </visual>
</toast>
"@
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("BarcodeHID").Show($toast)
`, title, body)
		_ = exec.Command("powershell", "-WindowStyle", "Hidden",
			"-NonInteractive", "-Command", script).Run()
	}
}

func notifyRunning(ip string, port int) {
	notify(
		"BarcodeHID is running",
		fmt.Sprintf("Open on your phone:\nhttps://%s:%d\n\nRight-click tray icon to show QR.", ip, port),
	)
}

func notifyAlreadyRunning(ip string, port int) {
	notify(
		"BarcodeHID is already running",
		fmt.Sprintf("Server at:\nhttps://%s:%d\n\nOpening QR code page…", ip, port),
	)
}
