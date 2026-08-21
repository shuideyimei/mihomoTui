# Quick Start Guide

This guide will help you install, configure, and get started with **mihomoTui**.

---

## 📋 Prerequisites

1. **Go 1.24+** (for building from source)
2. **A terminal with UTF-8, 256-color, and mouse support** (e.g. iTerm2, WezTerm, Alacritty, GNOME Terminal, Windows Terminal)
3. A running **[Mihomo](https://github.com/MetaCubeX/mihomo)** core instance with `external-controller` enabled.

### Check Terminal Environment

```bash
# Verify color support (should output >= 256)
echo $TERM
tput colors

# Ensure UTF-8 locale is set
echo $LANG
```

---

## 📥 Installation & Running

### 1. Clone the Repository

```bash
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui
```

### 2. Download Dependencies & Run

```bash
go mod tidy
go run main.go
```

### 3. Build Executable Binary

```bash
# Compile executable
go build -o mihomoTui .

# Run binary
./mihomoTui
```

---

## ⚡ First-Time Configuration

### Step 1: Launch and Configure API

When launching mihomoTui for the first time:
1. Press `F8` (or `Alt+8` / click **Settings** on the sidebar) to enter the Settings page.
2. In the **API URL** input, enter your Mihomo external controller address (default: `http://127.0.0.1:9090` or a Unix socket path such as `unix:///tmp/mihomo.sock`).
3. If Mihomo is configured with a secret, enter it into the **API Secret** field.
4. Click **Save** (or press `Enter`). Settings are saved to `~/.config/mihomoTui/config.json`.

### Step 2: Verify Connection & Switch Modes

Once saved, the header bar will show a green `● Connected` indicator with the core version, and the bottom status bar will display real-time traffic statistics.
- Press `F1` to view live throughput and memory usage on the **Dashboard**.
- Press `F2` to enter **Proxies**, where you can press `T` to run batch HTTP latency tests or press `R` (Rule), `G` (Global), or `D` (Direct) to switch proxy modes instantly.

---

## 🗺️ Page Navigation & Shortcut Cheatsheet

| Shortcut | Destination / Feature | Description |
|:---|:---|:---|
| `F1` / `Alt+1` | **Dashboard** | View throughput, memory usage, active connections, and toggle TUN / Allow LAN |
| `F2` / `Alt+2` | **Proxies** | Browse proxy groups and nodes, press `T` for latency speed tests, `/` to filter |
| `F3` / `Alt+3` | **Connections** | Inspect live connections, press `D` to close one, `C` to close all, `Space` to pause |
| `F4` / `Alt+4` | **Logs** | Stream real-time logs with level filters (Debug/Info/Warn/Error), `Space` to pause |
| `F5` / `Alt+5` | **Profiles** | Discover and hot-switch active configuration profiles; in-TUI YAML text editor |
| `F6` / `Alt+6` | **Subscriptions** | Import proxy subscriptions from remote URLs or local files with traffic stats |
| `F7` / `Alt+7` | **Config Editor** | Unified configuration editor: press `Tab` between **Proxy Groups**, **Rules**, **Providers** |
| `F8` / `Alt+8` | **Settings** | Configure API endpoints and switch UI language (zh/en) dynamically without restart |
| `:` | **Command Palette** | Open quick fuzzy-search modal to jump to any page or feature |
| `?` / `F12` | **Help Center** | Open interactive keyboard shortcut guide |
| `Esc` | **Sidebar Navigation** | Exit input focus and return to the left navigation sidebar |
| `Tab` / `Shift+Tab`| **Focus Cycling** | Cycle focus between form inputs, tables, and buttons |

---

## ⚙️ Configuration Files

### 1. mihomoTui App Configuration
File location: `~/.config/mihomoTui/config.json`
```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  },
  "language": "en"
}
```

### 2. Mihomo Core Configuration Discovery
mihomoTui discovers the running Mihomo `config.yaml` in the following precedence:
1. `MIHOMO_CONFIG_PATH` environment variable if set.
2. Parsing CLI arguments (`-d`, `-f`, `--config`) of the running Mihomo process.
3. Standard fallback locations (`~/.config/mihomo/config.yaml`, `/etc/mihomo/config.yaml`, `~/.config/clash/config.yaml`).

---

## 🛠️ Cross-Compilation

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-linux-amd64 .

# Linux arm64 (Raspberry Pi / Embedded)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-linux-arm64 .

# macOS ARM64 (Apple Silicon M1/M2/M3/M4)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-darwin-arm64 .

# Windows 64-bit
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-windows-amd64.exe .
```

---

## ❓ Troubleshooting

### 1. Header displays "Disconnected"
- Check that the Mihomo core process is running: `curl http://127.0.0.1:9090/version`
- Verify that the API URL and Secret configured in `F8 Settings` match the `external-controller` and `secret` in your Mihomo configuration.

### 2. Permission Denied when modifying configuration
- If your system Mihomo runs as root (e.g. `/etc/mihomo/config.yaml`), run with `sudo ./mihomoTui` or grant write access:
  ```bash
  sudo chown $USER ~/.config/mihomo/config.yaml
  ```
