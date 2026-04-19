# BarcodeHID — macOS Setup Guide

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

1. Double-click **BarcodeHID** in Applications (or click it in Finder)

2. A notification appears in the top-right corner of your screen:
   ```
   BarcodeHID is running
   Open on your phone: https://192.168.x.x:8765
   ```

3. A small **BarcodeHID icon** appears in your **menu bar** (top-right,
   near the clock). This means the app is running in the background.

> If you see **"BarcodeHID cannot be opened because it is from an
> unidentified developer"** → go back to Step 3 and make sure you ran
> the `xattr` command.

> If you see **"BarcodeHID" wants access to control your computer** →
> click **OK** and go back to Step 4 to grant Accessibility permission.

---

## Step 6 — Connect your phone

1. **Right-click** (or two-finger click) the BarcodeHID icon in the menu bar
2. Click **Show QR Code**
3. A page opens in your browser showing a QR code

4. On your phone:
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

## Menu bar controls

Right-click the BarcodeHID icon in the menu bar to:

| Option | What it does |
|---|---|
| **Connected / Not connected** | Shows current phone status |
| **Show QR Code** | Opens the pairing page in your browser |
| **Copy wss:// URL** | Copies the server address to clipboard |
| **Auto-start on login** | Toggle: start BarcodeHID automatically when you log in |
| **Quit BarcodeHID** | Stop the server and exit |

---

## Auto-start on login (optional)

If you want BarcodeHID to start automatically every time you log in:

1. Right-click the menu bar icon
2. Click **Auto-start on login** to enable it (checkmark appears)

To disable: click it again to uncheck.

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
