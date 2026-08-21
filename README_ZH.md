# mihomoTui - 终端下的桌面级 Mihomo 客户端

<p align="center">
  <img src="static/image.png" alt="mihomoTui Dashboard" width="900">
</p>

<p align="center">
  <strong>在终端中体验桌面级客户端的强大与优雅 — 基于 Go 1.24 + tview 构建的现代化 <a href="https://github.com/MetaCubeX/mihomo">Mihomo</a> 代理管理 TUI。</strong>
</p>

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg?style=flat-square" alt="License"></a>
  <a href="https://github.com/MetaCubeX/mihomo"><img src="https://img.shields.io/badge/Mihomo-Meta-blue?style=flat-square" alt="Mihomo Meta"></a>
  <a href="#界面预览"><img src="https://img.shields.io/badge/UI-tview%20%2B%20tcell-orange?style=flat-square" alt="TUI"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/Language-English-lightgrey?style=flat-square" alt="English README"></a>
</p>

---

## 🌟 核心特性

- 🖥️ **现代终端 UI 设计** — 基于 [tview](https://github.com/rivo/tview) 构建，支持响应式布局、平滑焦点切换、完备鼠标交互与 256 色 / TrueColor 高清显示。
- 🌐 **代理节点与测速管理** — 实时展示代理提供者（Providers）、代理分组（Groups）与节点表格，支持批量 HTTP Ping 延迟测试、快速节点切换及关键词实时过滤（`/`）。
- ⚡ **模式与核心状态切换** — 顶部及状态栏一键即时切换 `Rule`（规则）、`Global`（全局）、`Direct`（直连）代理模式，支持 TUN 模式与 Allow LAN 局域网共享切换。
- 📊 **系统监控与仪表盘** — 实时可视化上下行速率波动、系统内存占用（Inuse / OS Limit）、活跃连接计数与核心版本状态。
- 🔗 **活跃连接实时监控** — 查看所有 TCP / UDP 连接元数据（源/目标 IP、端口、匹配规则链、传输流量与持续时间），支持关键词快速搜索、单个连接强行阻断与一键关闭全部连接。
- 📝 **实时日志流式过滤** — SSE 实时流式捕获 Mihomo 运行日志，支持日志等级（`debug`, `info`, `warning`, `error`）筛选、滚动跟踪与实时暂停/继续。
- 🗂️ **多配置文件管理 (Profiles)** — 智能发现本地多份配置，支持即时无缝切换激活、新建、备份与内置编辑器在线编辑 YAML 文件（带未保存防丢保护）。
- 📥 **全能订阅导入与转换** — 支持从 Remote URL 或本地文件一键导入，兼容 **Clash/Mihomo YAML**、**Surge**、**Base64**、**V2Ray (vmess)**、**Shadowsocks (ss)**、**Trojan**、**Hysteria2 (hy2)** 及纯文本代理列表，自动解析提取 UserInfo 流量剩余与过期时间。
- 🛠️ **统一配置编辑器 (Editor)** — 在单个页面内通过 `Tab` / `Shift+Tab` 极速切换三大核心子模块：
  - **代理组 (Proxy Groups)**：可视化增删改查 `select`、`url-test`、`fallback`、`load-balance` 策略组，支持正则过滤节点与自定义健康检查。
  - **路由规则 (Rules)**：查看、新增（DOMAIN, DOMAIN-SUFFIX, IP-CIDR, GEOIP, GEOSITE, RULE-SET, MATCH 等）、删除、上下调整顺序（`Alt+↑`/`Alt+↓`/`M` 移动至指定位置）及批量粘贴导入。
  - **规则集提供者 (Rule Providers)**：管理 HTTP / 本地行为规则集（`domain`, `ipcidr`, `classical`），支持定时自动更新与预设快速导入。
- ⚙️ **统一系统与 API 设置** — 支持配置外部控制器 API Endpoint / Secret，内置 **中 / 英文 (i18n)** 动态热切换，无需重启即时刷新全局 UI。
- 🚀 **命令面板 (Command Palette)** — 任意页面轻按 `:` 键唤起快速搜索面板，支持模糊搜索页面与命令直达。
- ❓ **快捷键速查中心** — 任意页面按 `?` 或 `F12` 弹出全屏快捷键帮助与操作指南。
- 🛡️ **原子级安全存储** — 所有配置文件写入均采用原子替换（Unix UID/GID 权限保留机制），内建防 SSRF 安全防护。

---

## 📸 界面预览

<table align="center">
  <tr>
    <td align="center" width="50%">
      <strong>1. 仪表盘 (Dashboard - F1)</strong><br>
      <img src="picture/01-dashboard.png" alt="Dashboard" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>2. 代理节点管理 (Proxies - F2)</strong><br>
      <img src="picture/02-proxies.png" alt="Proxies" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <strong>3. 活跃连接监控 (Connections - F3)</strong><br>
      <img src="picture/03-connections.png" alt="Connections" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>4. 实时日志流 (Logs - F4)</strong><br>
      <img src="picture/04-logs.png" alt="Logs" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <strong>5. 配置文件管理 (Profiles - F5)</strong><br>
      <img src="picture/05-profiles.png" alt="Profiles" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>6. 订阅导入与解析 (Subscriptions - F6)</strong><br>
      <img src="picture/06-subscriptions.png" alt="Subscriptions" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <strong>7. 代理组配置 (Proxy Groups - F7)</strong><br>
      <img src="picture/07-editor-groups.png" alt="Proxy Groups" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>8. 路由规则管理 (Rules - F7)</strong><br>
      <img src="picture/08-editor-rules.png" alt="Rules" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <strong>9. 规则提供者 (Rule Providers - F7)</strong><br>
      <img src="picture/09-editor-providers.png" alt="Rule Providers" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>10. 系统设置与多语言 (Settings - F8)</strong><br>
      <img src="picture/10-settings.png" alt="Settings" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <strong>11. 命令面板 (Command Palette - :)</strong><br>
      <img src="picture/11-palette.png" alt="Command Palette" width="100%">
    </td>
    <td align="center" width="50%">
      <strong>12. 快捷键帮助中心 (Help - ? / F12)</strong><br>
      <img src="picture/12-help.png" alt="Help" width="100%">
    </td>
  </tr>
</table>

---

## ⌨️ 快捷键速查

### 全局导航快捷键

| 快捷键 | 功能 | 说明 |
|:---|:---|:---|
| `F1` / `Alt+1` / `Ctrl+1` | **仪表盘** | 查看流量、系统内存、代理模式与核心状态 |
| `F2` / `Alt+2` / `Ctrl+2` | **代理** | 查看/切换代理节点，批量 HTTP 测速 |
| `F3` / `Alt+3` / `Ctrl+3` | **连接** | 实时连接监控与单/多连接关闭 |
| `F4` / `Alt+4` / `Ctrl+4` | **日志** | 实时日志流展示、过滤与暂停 |
| `F5` / `Alt+5` / `Ctrl+5` | **配置文件** | 配置文件切换、新建与内置在线编辑 |
| `F6` / `Alt+6` / `Ctrl+6` | **订阅** | 订阅链接/本地文件导入与更新 |
| `F7` / `Alt+7` / `Ctrl+7` | **配置编辑** | 聚合编辑代理组、路由规则与规则提供者 |
| `F8` / `Alt+8` / `Ctrl+8` | **系统设置** | API Endpoint 设置与中英语言切换 |
| `:` | **命令面板** | 快速模糊搜索任意页面或功能 |
| `?` / `F12` | **帮助中心** | 显示 / 关闭快捷键指南弹窗 |
| `Esc` | **焦点返回** | 将焦点退出当前输入并返回左侧导航栏 |
| `Tab` / `Shift+Tab` | **切换焦点 / 子标签** | 在页面控件间切换，或在配置编辑器子标签间切换 |
| `Ctrl+C` / `Ctrl+Q` | **退出** | 优雅退出应用程序 |

### 模块专属快捷键

<details>
<summary><strong>🔹 路由规则页面 (Rules) 快捷键</strong></summary>

- `A`：弹出表单新增规则
- `D`：删除当前选中的规则（带确认对话框）
- `M`：将选中规则移动到指定行号
- `P`：粘贴批量导入多行规则文本
- `Alt+↑` / `Alt+↓`：将当前规则上移 / 下移一行
- `/`：快速聚焦到规则搜索过滤框
- `R`：重新从核心加载规则
</details>

<details>
<summary><strong>🔹 代理页面 (Proxies) 快捷键</strong></summary>

- `T`：对当前选中代理组或节点发起延迟测速
- `R`：切换为 Rule 规则模式
- `G`：切换为 Global 全局模式
- `D`：切换为 Direct 直连模式
- `/`：快速搜索过滤代理节点
</details>

<details>
<summary><strong>🔹 活跃连接页面 (Connections) 快捷键</strong></summary>

- `D`：关闭选中的单个连接
- `C` / `Shift+D`：关闭全部活跃连接
- `/`：快速搜索主机名、IP 或规则名
- `Space`：暂停 / 恢复实时连接列表自动刷新
</details>

---

## 🚀 快速开始

> 📖 更多详细安装指南与疑难解答请参阅 [**QUICKSTART_ZH.md**](QUICKSTART_ZH.md)。

### 前置要求

1. **Go 1.24.1+**（支持源码编译）
2. **支持 256 色 / UTF-8 与鼠标事件的终端**（如 iTerm2, WezTerm, Alacritty, Windows Terminal 等）
3. 正在运行且开启了 `external-controller` 的 **[Mihomo](https://github.com/MetaCubeX/mihomo)** 核心

### 一键安装与运行

```bash
# 1. 克隆代码仓库
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui

# 2. 安装依赖并编译运行
go mod tidy
go run main.go
```

### 从预编译二进制直接运行

```bash
# 本地快速构建
go build -o mihomoTui .
./mihomoTui
```

---

## 🛠️ 项目架构

项目采用清晰的 **三层分层架构 (3-Tier Layered Architecture)** 与 **事件驱动总线 (Event-Driven Architecture)**：

```
                      ┌────────────────────────────────────────┐
                      │               main / app               │
                      └───────────────────┬────────────────────┘
                                          │
        ┌─────────────────────────────────┼────────────────────────────────┐
        ▼                                 ▼                                ▼
┌────────────────┐               ┌────────────────┐               ┌────────────────┐
│  Presentation  │ ──(调用服务)──►│ Service Layer  │ ──(接口调用)──►│ Repository/Drv │
│  (tview Pages) │               │ (Rule/Group/..)│               │ (YAML / API)   │
└───────┬────────┘               └────────────────┘               └────────┬───────┘
        │ (订阅/发布)                                                      │ (原子写入)
        ▼                                                                  ▼
┌────────────────┐                                                ┌────────────────┐
│    EventBus    │ ◄───────────────────────────────────────────── │ Local Storage  │
│ (TopicProxy..) │                                                │ (config.yaml)  │
└────────────────┘                                                └────────────────┘
```

### 目录结构

```
mihomoTui/
├── main.go                          # 程序入口与日志生命周期管理
├── go.mod / go.sum                  # Go 模块定义
├── LICENSE                          # MIT 许可证
├── static/
│   └── image.png                    # README 首屏展示图
├── picture/                         # 各页面高分辨率预览截图
│
├── internal/
│   ├── app.go                       # 应用程序启动、依赖注入与生命周期协调
│   ├── hotkey.go                    # 全局键盘事件调度
│   ├── palette.go                   # 命令面板 (Command Palette) 实现
│   │
│   ├── api/                         # Mihomo REST API 与 SSE 流式通信客户端
│   ├── config/                      # 应用配置管理与低阶 YAML AST 操作
│   ├── events/                      # 线程安全解耦事件总线 (EventBus)
│   ├── i18n/                        # 国际化语言包 (zh / en 字典与查表器)
│   ├── models/                      # 核心领域实体定义 (RuleDisplay, GroupEntry...)
│   ├── parser/                      # 多协议订阅格式解析器 (Clash, Surge, V2Ray...)
│   ├── repository/                  # 数据持久层抽象接口与 LocalYAMLDriver 实现
│   ├── service/                     # 业务服务层 (Rule, ProxyGroup, Provider, Profile)
│   ├── subscription/                # 远程/本地订阅安全获取与 SSRF 防护
│   ├── types/                       # 标准配置文档模型 (ConfigDocument)
│   │
│   └── ui/                          # TUI 核心组件与页面
│       ├── ui.go                    # UI 线程调度器与全局 Modal 控制
│       ├── theme.go                 # 全局配色主题
│       ├── navigation.go            # 导航栏元数据与页面路由信息
│       ├── components/              # 通用组件 (Header, StatusBar, NavBar...)
│       └── pages/                   # 各功能页面实现 (Dashboard, Proxies, Editor...)
```

---

## 🤝 参与贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/AmazingFeature`
3. 提交你的修改：`git commit -m 'feat: add some amazing feature'`
4. 运行全量测试验证：`go test -race -count=1 ./...`
5. 推送到分支并提交 Pull Request

---

## 📄 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## 💖 鸣谢

- [Mihomo (MetaCubeX)](https://github.com/MetaCubeX/mihomo) — 强大的核心代理引擎
- [tview](https://github.com/rivo/tview) — 终端 UI 基础库
- [tcell](https://github.com/gdamore/tcell/v2) — 终端底层处理与模拟驱动
