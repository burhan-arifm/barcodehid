# BarcodeHID — macOS Setup Guide

> ⚠️ **Current macOS status**
>
> | Feature | Status |
> |---|---|
> | Barcode scanning | ✅ Works |
> | Keyboard simulation (HID) | ✅ Works |
> | Background / tray mode | ⚠️ Not yet supported |
>
> The app runs in foreground (terminal) mode on macOS.
> Tray support is planned for a future release.

This guide walks you through installing and running BarcodeHID on macOS
from scratch. No technical experience required.

---

## What you need

- A Mac running macOS 12 (Monterey) or newer
- An Android or iPhone with a camera
- Both your Mac and phone on the same Wi-Fi network

---

## Step 1 — Download the app

1. Go to the [Releases page](../../releases/latest)
2. Under **Assets**, find the right file for your Mac:
   - **Apple Silicon** (M1, M2, M3, M4 chip) → `BarcodeHID-macos-arm64.app.zip`
   - **Intel chip** → `BarcodeHID-macos-amd64.app.zip`
   - Not sure which? Click **Apple menu () → About This Mac**. Look for "Chip" or "Processor".
     - Says "Apple M..." → download `arm64`
     - Says "Intel" → download `amd64`
3. Click the file to download it

---

## Step 2 — Unzip the app

1. Open **Finder** and go to your **Downloads** folder
2. Double-click `BarcodeHID-macos-arm64.app.zip` (or amd64)
3. A file called **BarcodeHID.app** appears in the same folder
4. Drag **BarcodeHID.app** to your **Applications** folder
   - Open a new Finder window → click **Applications** in the sidebar
   - Drag the app in

---

## Step 3 — Remove the quarantine flag

macOS automatically blocks apps downloaded from the internet that are
not signed by an Apple-registered developer. BarcodeHID is open source
and unsigned, so you need to tell macOS it is safe to run.

**This is a one-time step. You only do this once.**

1. Open **Terminal**
   - Press `Cmd + Space`, type `Terminal`, press `Enter`

2. Copy and paste this command, then press `Enter`:
   ```bash
   xattr -d com.apple.quarantine /Applications/BarcodeHID.app
   ```
   > If you see `No such xattr: com.apple.quarantine` — that's fine,
   > it just means the flag wasn't there. Continue to the next step.

3. You can now close Terminal.

---

## Step 4 — Grant Accessibility permission

BarcodeHID needs **Accessibility** permission to simulate keyboard input
when a barcode is scanned. macOS requires you to grant this manually.

1. Try opening the app first:
   - Go to **Applications** in Finder
   - Double-click **BarcodeHID**
   - A notification should appear saying **"BarcodeHID is running"**
   - If nothing happens, or you see an error — continue with steps below

2. Open **System Settings**
   - Click the Apple menu () → **System Settings**

3. Go to **Privacy & Security → Accessibility**

4. Click the **lock icon** at the bottom and enter your password to make changes

5. Click the **+** button

6. Navigate to your **Applications** folder and select **BarcodeHID**

7. Make sure the **toggle next to BarcodeHID is ON** (green)

8. Close System Settings

---

## Step 5 — Run the app

Open **Terminal** and run:

```bash
/Applications/BarcodeHID.app/Contents/MacOS/barcodehid
```

You will see a QR code printed in the terminal along with the server URL:

```
  Scanner    https://192.168.x.x:8765
  WebSocket  wss://192.168.x.x:8765
  ...

  Scan with your phone camera:
  [QR code appears here]
```

> If you see **"Accessibility permission required"** → go back to Step 4.

> If you see **"BarcodeHID cannot be opened"** → go back to Step 3 and
> make sure you ran the `xattr` command.

Keep this terminal window open while using BarcodeHID.
Press `Ctrl+C` to stop the server.

> **Note:** Background/tray mode is not yet supported on macOS.
> The terminal window must stay open while the app is running.
> This limitation will be addressed in a future release.

---

## Step 6 — Connect your phone

1. On your phone:
   - Open the **camera app**
   - Point it at the QR code on your screen
   - Tap the notification/link that appears
   - Your phone's browser opens

5. **First time only:** your browser shows a security warning
   - **Safari on iPhone:** tap **Show Details → visit this website → Visit Website**
   - **Chrome on Android:** tap **Advanced → Proceed to 192.168.x.x (unsafe)**
   - This warning is expected — BarcodeHID uses a self-signed certificate
     for local HTTPS. It is safe to proceed.

6. Your phone asks for **camera permission** → tap **Allow**

7. The scanner is now active. You should see your phone's camera viewfinder.

---

## Step 7 — Scan a barcode

1. Click into any text field on your Mac (a browser address bar, a document,
   a search box — anywhere you would normally type)
2. Hold a barcode in front of your phone camera
3. The barcode value is typed into your Mac automatically

---

## Auto-start on login (optional)

Since background/tray mode is not yet supported, auto-start on macOS
is not available. You will need to start BarcodeHID manually from
Terminal each time.

Once tray mode is supported in a future release, auto-start will be
added as a menu option.

---

## Troubleshooting

**App opens but nothing happens (no notification, no menu bar icon)**
→ Accessibility permission is missing. Go to Step 4.
→ Also try: right-click BarcodeHID.app → **Open** → click **Open** in the dialog.

**"BarcodeHID is damaged and can't be opened"**
→ The quarantine flag is still there. Go to Step 3 and run the `xattr` command again.

**Phone shows "Connection failed"**
→ Make sure your phone and Mac are on the same Wi-Fi network.
→ Try turning Wi-Fi off and on again on your phone.

**Phone camera opens but barcodes don't scan**
→ Make sure there's good lighting and the barcode fills most of the viewfinder.
→ Try moving slightly closer or further from the barcode.

**Nothing gets typed on the Mac after scanning**
→ Make sure a text field on your Mac is focused (clicked into) before scanning.
→ Check Accessibility permission is still enabled (System Settings → Privacy & Security → Accessibility).

**The menu bar icon disappeared**
→ BarcodeHID may have quit. Open it again from Applications.
→ Check if it is already running: right-click → you should see a notification
  saying "BarcodeHID is already running" if it is.

---

## Uninstalling

1. Right-click the menu bar icon → **Quit BarcodeHID**
2. Drag **BarcodeHID.app** from Applications to Trash
3. To remove auto-start: delete `~/Library/LaunchAgents/io.barcodehid.plist`
4. To remove the lock file: delete `/tmp/barcodehid.lock`
