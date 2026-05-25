# mihomoTui - TUI for Mihomo

> Provide a desktop client experience in the terminal — a modern terminal UI for managing the [Mihomo](https://github.com/MetaCubeX/mihomo) proxy.

[![License](https://img.shields.io/github/license/FlySky-z/mihomoTui)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/FlySky-z/mihomoTui?style=social)](https://github.com/FlySky-z/mihomoTui/stargazers)
[![Go Version](https://img.shields.io/badge/Go-1.24.1-00ADD8)](https://go.dev)

English | [简体中文](README_ZH.md)

![mihomoTui Demo](static/image.png)

---

## Features

- 🖥️ **Modern Terminal UI** — Beautiful interface built with [tview](https://github.com/rivo/tview)
- 🌐 **Proxy Management** — Browse and switch proxy nodes and proxy groups in real time
- ⚙️ **Configuration Control** — Toggle TUN mode, Allow LAN, and proxy mode on the fly
- 📊 **Real-time Monitoring** — Live traffic statistics (upload/download speed), memory usage, and active connection count
- 📋 **Rule Management** — View, add, edit, delete, reorder, and batch-import proxy rules and RULE-SET rules
- 🔖 **Rule Providers** — Add, update, and delete rule-provider subscriptions (HTTP/local)
- 📥 **Subscription Management** — Import proxy subscriptions from URL or local file, merge into your existing config
- 📦 **Proxy Group Management** — Create, update, and delete custom proxy groups with types: `select`, `url-test`, `fallback`, `load-balance`
- 📝 **Real-time Logs** — Live log streaming with level filtering and pause/resume
- 🔗 **Connection Monitoring** — View active connections, close individual or all connections, auto-refresh
- 🖱️ **Full Mouse Support** — Click to navigate, switch tabs, toggle buttons
- ⌨️ **Keyboard Shortcuts** — F-keys, Ctrl+number, Alt+letter for quick navigation
- 🔄 **Config Hot-Reload** — Modify mihomo config.yaml directly from the TUI with rollback on failure
- 📎 **Smart Config Merging** — Merge subscription configs with your local config (proxies, groups, rules) with conflict resolution

## Project Status

**Current Version**: `v0.0-Alpha` — In Development

### Development Progress

- [x] API client development
- [x] UI framework setup and layout
- [x] Core functionality implementation
- [x] Proxy management (view & switch)
- [x] Real-time traffic monitoring
- [x] Real-time log streaming
- [x] Connection monitoring
- [x] Dashboard with mode/config controls
- [x] App configuration (API URL & Secret)
- [x] Rule management (view, add, edit, delete, batch import)
- [x] Rule providers management
- [x] Subscription import & config merging
- [x] Proxy group management (CRUD)
- [x] Config hot-reload with rollback
- [x] CI/CD — Cross-platform builds (Linux, macOS, Windows)
- [ ] Check config details
- [ ] Switch config files
- [ ] Modify ports from UI
- [ ] Multi-language support (i18n)

## Tech Stack

| Layer      | Technology                                      |
|------------|-------------------------------------------------|
| Language   | Go 1.24.1                                       |
| UI Toolkit | [tview](https://github.com/rivo/tview)          |
| Terminal   | [tcell](https://github.com/gdamore/tcell/v2)    |
| Config     | JSON (app) + YAML (mihomo)                      |
| API        | Mihomo REST API (HTTP)                          |
| Streaming  | Mihomo SSE/chunked endpoints (traffic, logs, memory) |
| CI/CD      | GitHub Actions (cross-platform builds, releases)       |

## Project Structure

```
mihomoTui/
├── main.go                          # Program entry point
├── go.mod / go.sum                  # Go module definition
├── LICENSE                          # MIT License
├── .gitignore
├── static/
│   └── image.png                    # Demo screenshot
│
├── .github/workflows/
│   ├── ci-build.yml                 # CI: build check on push to main
│   └── build-release.yml            # CD: cross-platform release on tag push
│
├── internal/
│   ├── app.go                       # Application orchestration, page wiring, layout
│   ├── hotkey.go                    # Global keyboard shortcuts (F-keys, Ctrl, Alt)
│   │
│   ├── api/
│   │   ├── client.go                # HTTP API client for all Mihomo endpoints
│   │   └── stream.go                # SSE streaming client (traffic, logs, memory)
│   │
│   ├── cache/
│   │   ├── cache.go                 # In-memory TTL cache (thread-safe)
│   │   └── cache_test.go
│   │
│   ├── config/
│   │   ├── config.go                # App config manager (JSON-based, save/load/backup)
│   │   └── mihomo_config.go         # Mihomo YAML config CRUD (groups, rules, providers)
│   │
│   ├── event/
│   │   ├── bus.go                   # Pub/sub event bus
│   │   └── bus_test.go
│   │
│   ├── hotreload/
│   │   └── manager.go               # Config hot-reload with rollback support
│   │
│   ├── merge/
│   │   ├── engine.go                # Smart config merging engine
│   │   └── engine_test.go
│   │
│   ├── models/
│   │   └── clash.go                 # Data models: Config, Proxy, Connection, Rule, etc.
│   │
│   ├── parser/
│   │   ├── parser.go                # YAML config parser (Clash/Mihomo format)
│   │   ├── parser_test.go
│   │   ├── strutil.go               # String utility helpers
│   │   ├── util.go                  # Parsing utilities
│   │   └── validate.go              # Validation helpers
│   │
│   ├── proxygroup/
│   │   ├── manager.go               # Proxy group & dependency management
│   │   └── manager_test.go
│   │
│   ├── runtime/
│   │   └── state.go                 # Runtime state container (bus, cache, managers)
│   │
│   ├── subscription/
│   │   └── manager.go               # Subscription import from URL or local file
│   │
│   ├── types/
│   │   ├── config.go                # ConfigDocument — canonical parsed config
│   │   ├── event.go                 # Event types (subscription, config, proxy, cache)
│   │   ├── merge.go                 # Merge strategy & result types
│   │   ├── misc.go                  # UpdateTask, CacheEntry, ReloadEvent
│   │   ├── proxygroup.go            # ProxyGroupDef, ImportResult, ConflictEntry
│   │   └── subscription.go          # Subscription & UserInfo
│   │
│   ├── updater/
│   │   └── manager.go               # Background update task manager
│   │
│   ├── ui/
│   │   ├── ui.go                    # UI updater singleton (QueueUpdate wrapper)
│   │   │
│   │   ├── components/
│   │   │   ├── header.go            # Top header bar (app name, version, connection status)
│   │   │   ├── sidebar.go           # Navigation sidebar (9 pages)
│   │   │   ├── statusbar.go         # Bottom status bar (mode, traffic, help text)
│   │   │   └── buttonnav.go         # Button navigation primitives
│   │   │
│   │   └── pages/
│   │       ├── pages.go             # Page constructors & ActivatablePage interface
│   │       ├── dashboard.go         # Dashboard: connections, system info, TUN/AllowLAN controls
│   │       ├── proxies.go           # Proxies page: provider list → group → node table + delay test
│   │       ├── connections.go       # Connections page: live connection table, close/search
│   │       ├── config.go            # Config page: API URL & Secret form
│   │       ├── logs.go              # Logs page: real-time streaming with pause/resume
│   │       ├── rules.go             # Rules page: view/add/edit/delete/batch-import rules
│   │       ├── settings.go          # Settings page (WIP)
│   │       ├── proxygroups/         # Proxy groups management: CRUD, type selection
│   │       ├── ruleproviders/       # Rule providers management: add/update/delete
│   │       ├── subscriptions/       # Subscription management: import, merge preview
│   │       ├── mergepreview/        # Config merge preview UI
│   │       └── proxyimport/         # Proxy import UI
│   │
│   └── utils/
│       ├── convert.go               # FormatBytes — human-readable byte formatting
│       └── env.go                   # Environment variable helper
```

## Pages Overview

| # | Page             | Description |
|---|------------------|-------------|
| 1 | **Dashboard**    | Connection stats, system memory, proxy mode, TUN/AllowLAN toggle buttons |
| 2 | **Proxies**      | Provider list → group list → node table with delay testing |
| 3 | **Connections**  | Live connection table with search, close individual/all |
| 4 | **Config**       | Mihomo API base URL and Secret configuration |
| 5 | **Logs**         | Real-time log streaming with pause/resume and level filtering |
| 6 | **Subscriptions**| Import proxy subscriptions from URL or file, merge into config |
| 7 | **Proxy Groups** | Create, update, delete custom proxy groups |
| 8 | **Rules**        | View, add, edit, delete, reorder (↑/↓/move-to), bulk-import proxy rules |
| 9 | **Rule Providers** | Add, update, delete rule-provider subscriptions |

## Keyboard Shortcuts

| Key              | Action                     |
|------------------|----------------------------|
| `F1` / `Ctrl+1` | Dashboard                  |
| `F2` / `Ctrl+2` | Proxies                    |
| `F3` / `Ctrl+3` | Connections                |
| `F4` / `Ctrl+4` | Config                     |
| `F5` / `Ctrl+5` | Logs                       |
| `F6` / `Ctrl+6` | Subscriptions              |
| `F7` / `Ctrl+7` | Proxy Groups               |
| `Ctrl+C` / `Ctrl+Q` | Quit application       |
| `Esc`            | Return focus to sidebar    |
| `Alt+D` / `Alt+P` / `Alt+R` / `Alt+C` / `Alt+L` / `Alt+S` / `Alt+G` | Switch to page by letter |
| `Tab` / `Shift+Tab` | Cycle focus between controls |
| `Ctrl+R`        | Refresh current page data  |

### Rules Page Shortcuts

| Key              | Action                     |
|------------------|----------------------------|
| `A`             | Add rule                   |
| `D`             | Delete selected rule       |
| `R`             | Refresh rules              |
| `P`             | Paste import rules         |
| `M`             | Move selected rule to position |
| `↑`/`↓`        | Navigate rule list         |
| `Alt+↑`         | Move selected rule up      |
| `Alt+↓`         | Move selected rule down    |
| `/`             | Focus search filter        |

> Mouse navigation is also fully supported — click sidebar items, buttons, and table cells.

## Quick Start

### Prerequisites

- Go 1.24.1 or higher
- A terminal with 256-color and mouse support
- A running [Mihomo](https://github.com/MetaCubeX/mihomo) instance with external-controller enabled

### Install & Run

```bash
# Clone the repository
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui

# Install dependencies
go mod tidy

# Run the application
go run main.go
```

### First-Time Setup

1. Start the application
2. Navigate to the **Config** page (click sidebar or press `F4`/`Ctrl+4`)
3. Enter your Mihomo API address (default: `http://127.0.0.1:9090`)
4. Enter your API Secret (if configured in Mihomo)
5. Save — config is written to `~/.config/mihomoTui/config.json`
6. **Restart the application** for the settings to take effect

The status bar at the bottom will show the current proxy mode and real-time traffic once connected.

## Configuration

### App Config (`~/.config/mihomoTui/config.json`)

```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  }
}
```

### Mihomo Config

The TUI can directly modify your running Mihomo config.yaml — it locates it by:
1. Scanning `/proc` for running mihomo processes and their `-d` / `--directory` flags
2. Falling back to common locations: `/etc/mihomo/config.yaml`, `~/.config/mihomo/config.yaml`

Operations on the mihomo config (proxy groups, rules, providers) require **sudo** access to write to the config file, as it is typically owned by root.

## Development

### Build from Source

```bash
# Build for current platform
go build -o mihomoTui .

# Cross-compile (Linux amd64)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-linux-amd64 .

# Cross-compile (macOS ARM64)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-darwin-arm64 .

# Cross-compile (Windows)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-windows-amd64.exe .
```

### Run Tests

```bash
go test ./internal/...
```

## Contributing

Issues and Pull Requests are welcome!

### Development Workflow

1. Fork the project
2. Create your feature branch: `git checkout -b feature/AmazingFeature`
3. Commit your changes: `git commit -m 'Add some AmazingFeature'`
4. Push to the branch: `git push origin feature/AmazingFeature`
5. Open a Pull Request

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Mihomo](https://github.com/MetaCubeX/mihomo) — The core proxy engine
- [tview](https://github.com/rivo/tview) — Powerful terminal UI library for Go
- [tcell](https://github.com/gdamore/tcell/v2) — Low-level terminal handling

## Contact

- Submit an [Issue](https://github.com/shuideyimei/mihomoTui/issues)
- Start a [Discussion](https://github.com/shuideyimei/mihomoTui/discussions)
