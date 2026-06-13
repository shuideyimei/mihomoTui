# Quick Start

## Prerequisites

- **Go 1.24.1** or higher
- A terminal with **256-color** and **mouse support** (GNOME Terminal, iTerm2, Windows Terminal, etc.)
- A running [Mihomo](https://github.com/MetaCubeX/mihomo) instance with `external-controller` enabled

### Verify Your Terminal

```bash
# Check terminal color count (should be ≥ 256)
echo $TERM
tput colors

# Mouse events are supported by most modern terminals by default
```

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Build and Run

```bash
# Run directly (development mode)
go run main.go

# Or build a binary
go build -o mihomoTui .
./mihomoTui
```

## First-Time Setup

### Step 1: Launch the Application

```bash
go run main.go
```

You should see an interface similar to this (navigation bar at top, page content in the middle, status bar at bottom):

```
┌─ mihomoTui v0.0-Alpha ────────────────────────── Status: Connected ─┐
│ [Dashboard] [Proxies] [Connections] [Config] [Logs] ...             │
│ ┌──────────────────────────────────────────────────────────────────┐ │
│ │                    Welcome to mihomoTui                          │ │
│ │              Please configure API URL and Secret first           │ │
│ └──────────────────────────────────────────────────────────────────┘ │
│ Mode: Rule   ↑ 0B/s  ↓ 0B/s  Memory: 0MB  Connections: 0          │
└──────────────────────────────────────────────────────────────────────┘
```

### Step 2: Configure API Connection

1. Press `F4` or click **Config** on the navigation bar
2. Enter your Mihomo API address (default: `http://127.0.0.1:9090`)
3. Enter your API Secret (if configured in your Mihomo `config.yaml`)
4. Click **Save** (or press `Enter`)
5. Config is automatically written to `~/.config/mihomoTui/config.json`
6. **Restart the application** for the settings to take effect

### Step 3: Verify Connection

After restarting, the status bar at the bottom should display:
- Current proxy mode (Rule / Global / Direct)
- Real-time traffic (upload/download speed)
- Memory usage
- Active connection count

If the status bar shows "Disconnected", check:
- Whether Mihomo is running
- Whether the API URL and Secret are correct
- Whether the port is accessible

## Configuration Details

### App Configuration

Config file location: `~/.config/mihomoTui/config.json`

```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  }
}
```

| Field | Description |
|-------|-------------|
| `base_url` | Mihomo API address, format `http://ip:port` |
| `secret` | API secret, must match the `secret` field in `config.yaml` |

### Mihomo Configuration Requirements

Your Mihomo `config.yaml` must have `external-controller` enabled:

```yaml
# config.yaml example
port: 7890
socks-port: 7891
external-controller: 0.0.0.0:9090  # required
secret: ""                          # optional, recommended
```

### Finding the Mihomo Config File

The TUI locates your Mihomo `config.yaml` by:

1. Using `MIHOMO_CONFIG_PATH` when explicitly set
2. Scanning `/proc` for running mihomo processes and their `-d` / `--directory` flags
3. Falling back to common locations:
   - `/etc/mihomo/config.yaml`
   - `~/.config/mihomo/config.yaml`

> ⚠️ Write operations on the Mihomo config (proxy groups, rules, providers) typically require `sudo` access, as the config file is usually owned by root.

## Page Navigation

| Key | Navigate To |
|-----|-------------|
| `F1` / `Ctrl+1` / `Alt+D` | Dashboard |
| `F2` / `Ctrl+2` / `Alt+P` | Proxies |
| `F3` / `Ctrl+3` / `Alt+R` | Connections |
| `F4` / `Ctrl+4` / `Alt+C` | Config |
| `F5` / `Ctrl+5` / `Alt+L` | Logs |
| `F6` / `Ctrl+6` / `Alt+S` | Subscriptions |
| `F7` / `Ctrl+7` / `Alt+G` | Proxy Groups |
| `F8` / `Ctrl+8` / `Alt+U` | Rules |
| `F9` / `Ctrl+9` / `Alt+T` | Rule Providers |
| `F10` / `Ctrl+0` / `Alt+K` | Settings |
| `F11` / `Alt+M` | Config Manager |
| `?` | Show/hide keyboard shortcuts help |

Mouse click on navigation bar items is also supported.

## Building from Source

### Build for Current Platform

```bash
go build -o mihomoTui .
```

### Cross-Compile

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-linux-amd64 .

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-darwin-arm64 .

# macOS amd64 (Intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-darwin-amd64 .

# Windows amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-windows-amd64.exe .
```

### Run Tests

```bash
# Run all internal package tests
go test ./internal/...

# Run tests for a specific package
go test ./internal/ui/pages/...
go test ./internal/api/...
```

## Troubleshooting

### No Display / Garbled Characters After Launch

Make sure your terminal:
- Supports UTF-8 encoding
- Has `TERM=xterm-256color` or similar set
- Uses a monospace font

### Status Bar Shows "Disconnected"

1. Verify Mihomo is running: `ps aux | grep mihomo`
2. Check API accessibility: `curl http://127.0.0.1:9090/version`
3. Verify the address and secret in the Config page

### Cannot Save Configuration

The config directory `~/.config/mihomoTui/` is created automatically. If saving fails, check:
- Directory permissions
- Disk space availability

### Modifying Mihomo Config Requires sudo

```bash
# Run with elevated privileges temporarily
sudo ./mihomoTui

# Or change config file ownership (not recommended)
sudo chown $USER /etc/mihomo/config.yaml
```
