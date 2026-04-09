# DroidLink

Robust WiFi ADB for Android developers. DroidLink keeps your wireless debugging connection alive — auto-reconnecting on drops, discovering devices via mDNS, and queuing installs until your device is back online.

> Inspired by Xcode's seamless wireless install experience. Android devs deserve the same.

---

## The Problem

Android Studio's WiFi debugging drops constantly. Every time your screen locks, Doze mode kicks in and kills the WiFi connection. You run `adb connect` again. And again. It breaks your flow.

## What DroidLink Does

- **Auto-reconnects** — heartbeat every 4s, reconnects within seconds of a drop
- **mDNS discovery** — finds your device on the network automatically, no IP hunting
- **Install queue** — `droidlink install app.apk` waits for reconnection if the device is mid-drop, then installs
- **Companion app** *(optional)* — a tiny Android foreground service that holds a `WifiLock`, preventing Doze from killing the connection in the first place
- **Persistent pairing** — pair once, works forever across reboots

---

## Prerequisites

Before installing DroidLink, make sure you have:

**1. ADB (Android Debug Bridge)**

| Platform | Command |
|---|---|
| macOS | `brew install android-platform-tools` |
| Windows | `scoop install adb` |
| Linux | `sudo apt install adb` |
| Manual | [Download Platform Tools](https://developer.android.com/tools/releases/platform-tools) → extract → add to PATH |

Verify: `adb version`

**2. Android device running Android 11+** (API 30) with Developer Options enabled

**3. Both your device and computer on the same WiFi network**

**4. Companion app** *(optional, recommended)*

A tiny Android app that holds a WiFi lock to prevent connection drops. Without it DroidLink still works — it just auto-reconnects on drops instead of never dropping. See [Companion App](#companion-app-optional-recommended) for details.

---

## Installation

### macOS — Homebrew (recommended)

```sh
brew tap Rohit-554/tap
brew install droidlink
```

### macOS / Linux — curl

```sh
curl -fsSL https://raw.githubusercontent.com/Rohit-554/droidLink/HEAD/scripts/install.sh | sh
```

### Windows — Scoop

```sh
scoop bucket add droidlink https://github.com/Rohit-554/scoop-bucket
scoop install droidlink
```

### Build from source

Requires Go 1.21+.

```sh
git clone https://github.com/Rohit-554/droidLink.git
cd droidLink
go build -o droidlink ./cmd/droidlink
sudo mv droidlink /usr/local/bin/
```

---

## Quick Start

**1. Enable Wireless Debugging on your Android device**

`Settings → Developer Options → Wireless Debugging` → toggle on

**2. Pair your device**

```sh
droidlink pair
```

DroidLink scans your network, finds the device showing a pairing code, prompts for the 6-digit PIN, and saves the pairing.

**3. Start the daemon**

```sh
droidlink start
```

The daemon runs in the background, maintains the connection, and auto-reconnects on drops.

**4. Install an APK**

```sh
droidlink install app-debug.apk
```

---

## Commands

```
droidlink pair                  Pair a new Android device over WiFi
droidlink devices               List paired devices and their connection status
droidlink install <apk>         Install an APK on all paired devices
droidlink start                 Start the DroidLink daemon
droidlink stop                  Stop the DroidLink daemon
droidlink unpair <serial>       Remove a paired device
droidlink --version             Show version info
droidlink --help                Show help
```

### `droidlink pair`

Scans for devices advertising a pairing code via mDNS (`_adb-tls-pairing._tcp`). Prompts for the 6-digit PIN shown on the device screen, completes pairing, connects, and saves the device to `~/.droidlink/devices.json`.

```sh
$ droidlink pair

On your Android device:
  Settings → Developer Options → Wireless Debugging → Pair device with pairing code

Scanning for devices showing a pairing code...

Found 1 device(s):
  [1] pixel6.local. (192.168.1.42:38517)

Enter the 6-digit pairing code shown on your device: 123456
Pairing with 192.168.1.42:38517...
Paired successfully.
Connecting to 192.168.1.42:5555...
✓ Pixel 6 (192.168.1.42:5555) paired and saved.
```

### `droidlink devices`

```sh
$ droidlink devices

  192.168.1.42:5555         192.168.1.42:5555  connected
  192.168.1.77:5555         192.168.1.77:5555  reconnecting
```

### `droidlink install <apk>`

Installs on all paired devices. If a device is mid-reconnect, the install waits up to 30 seconds for it to come back before retrying.

```sh
$ droidlink install app-debug.apk

✓ Installed on 192.168.1.42:5555
✓ Installed on 192.168.1.77:5555
```

---

## Companion App (Optional, Recommended)

The companion app is **not required** — DroidLink works without it. However, without it your device may drop the WiFi connection when the screen locks or Android's Doze mode kicks in.

**What it does:** A tiny Kotlin Android foreground service (~100 lines) that holds a `WifiLock` (`WIFI_MODE_FULL_HIGH_PERF`), telling Android to keep the WiFi chip fully active regardless of screen state or battery optimization.

| Without companion app | With companion app |
|---|---|
| Connection drops when screen locks | Connection stays alive indefinitely |
| DroidLink auto-reconnects (takes ~8s) | No drops, no reconnects needed |
| Works fine for most use cases | Best for long builds or overnight deployments |

**Install the companion app:**

1. Open `companion/` in Android Studio
2. Build and run on your device: `Run → Run 'app'`
3. The app starts automatically on boot and shows a persistent notification while active

**Permissions used:** `FOREGROUND_SERVICE`, `ACCESS_WIFI_STATE`, `CHANGE_WIFI_STATE` — nothing else. No internet, no data collection.

---

## How It Works

```
┌─────────────────────────────────────────────────────┐
│                    droidlink CLI                     │
│  pair · devices · install · start · stop · unpair   │
└────────────────────┬────────────────────────────────┘
                     │ Unix socket IPC (~/.droidlink/daemon.sock)
┌────────────────────▼────────────────────────────────┐
│                  DroidLink Daemon                    │
│                                                      │
│  ┌─────────────────┐    ┌──────────────────────┐    │
│  │  mDNS Discovery │    │   Connection Manager  │    │
│  │  (zeroconf)     │    │                       │    │
│  └─────────────────┘    │  ping every 4s        │    │
│                         │  3 misses → reconnect │    │
│  ┌─────────────────┐    └──────────────────────┘    │
│  │  Device Store   │                                 │
│  │  (~/.droidlink/ │    ┌──────────────────────┐    │
│  │   devices.json) │    │   Install Queue       │    │
│  └─────────────────┘    │   (waits for device)  │    │
│                         └──────────────────────┘    │
└─────────────────────────────────────────────────────┘
                     │
              adb binary
                     │
┌────────────────────▼────────────────────────────────┐
│              Android Device                          │
│                                                      │
│   WifiLockService (companion app)                    │
│   └── holds WIFI_MODE_FULL_HIGH_PERF lock            │
│       prevents Doze from cutting WiFi                │
└─────────────────────────────────────────────────────┘
```

### Reconnect flow

1. Heartbeat pings the device every **4 seconds** via `adb shell echo ping`
2. After **3 consecutive misses**, the daemon marks the device as `reconnecting`
3. `adb connect` is retried every **2 seconds** for up to **30 seconds**
4. Any `droidlink install` commands queued during reconnection wait and resume automatically once the device is back

---

## Configuration

DroidLink stores all state in `~/.droidlink/`:

```
~/.droidlink/
├── devices.json      — paired device registry
└── daemon.sock       — IPC socket (created when daemon starts)
```

No config file needed — everything is driven by the CLI.

---

## Contributing

```sh
git clone https://github.com/Rohit-554/droidLink.git
cd droidLink
go test ./...
go build ./cmd/droidlink
```

PRs welcome. Please run `go test ./...` before submitting.

---

## License

MIT — see [LICENSE](LICENSE)
