# Changelog

All notable changes to the **mihomoTui** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-08-21

### 🎉 Initial Release

Welcome to the first official release of **mihomoTui** — A modern, full-featured terminal UI client for Mihomo (Clash.Meta) built with Go and tview.

### ✨ Key Features

#### 1. Dashboard & System Monitoring (`F1`)
- Real-time upstream / downstream traffic rate visualization and memory usage tracking.
- One-click toggles for proxy running modes (`Rule`, `Global`, `Direct`), `Allow LAN`, and `TUN` mode.
- Core connection status and version indicator with auto-reconnection.

#### 2. Proxies & Latency Testing (`F2`)
- Multi-group and multi-provider node browsing with real-time latency badges.
- Batch HTTP ping testing (Space) and single-node delay re-test (R).
- Instant keyword filtering (`/`) and keyboard navigation.

#### 3. Real-Time Connections Monitor (`F3`)
- Live connection inspector with metadata (source/destination IP, port, matched rule, transferred payload, duration).
- Quick search, single connection termination, and close-all connections (`D` / `Del`).

#### 4. Streaming Log Viewer (`F4`)
- Real-time SSE streaming logs with level filtering (`debug`, `info`, `warning`, `error`).
- Pause / resume streaming (`Space`), log clearing (`C`), and search (`/`).

#### 5. Profile Management (`F5`)
- Local profile discovery, seamless activation, creation, and backup.
- Built-in configuration file editor with unsaved changes protection.

#### 6. Universal Subscription Manager (`F6`)
- Multi-protocol parser supporting **Clash/Mihomo YAML**, **Surge**, **Base64**, **V2Ray (vmess)**, **Shadowsocks (ss)**, **Trojan**, **Hysteria2 (hy2)**, and raw proxy lists.
- UserInfo quota & expiration date extraction and visual progress bars.

#### 7. Unified Configuration Editor (`F7`)
- **Proxy Groups**: Visual management of `select`, `url-test`, `fallback`, and `load-balance` groups.
- **Routing Rules**: View, insert, delete, reorder (`Alt+↑`/`Alt+↓`), and batch paste rule lists.
- **Rule Providers**: Manage behavioral rule sets (`domain`, `ipcidr`, `classical`) with auto-update intervals.

#### 8. Settings & Dynamic Internationalization (`F8`)
- External controller endpoint & secret management.
- Dynamic runtime language switching between **Simplified Chinese** (`zh`) and **English** (`en`) without restarting.

#### 9. Global Power Tools
- **Command Palette (`:`)**: Fast fuzzy finder to jump across all pages and actions.
- **Help Center (`?` / `F12`)**: Full-screen shortcut reference guide.
- **CLI Flags**: Support for `-v` / `--version` and `-h` / `--help`.

#### 10. Robust Architecture
- High-concurrency SSE stream safety with zero UI freeze.
- Atomic file writes with Unix permissions preservation.
- Full E2E terminal testing integration via `mcp-tui-test`.
