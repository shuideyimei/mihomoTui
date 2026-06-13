# mihomoTui - Mihomo 终端界面

> 在终端中提供桌面级客户端体验 — 基于 Go 和 tview 构建的 [Mihomo](https://github.com/MetaCubeX/mihomo) 代理管理 TUI。

[![Go Version](https://img.shields.io/badge/Go-1.24.1-00ADD8)](https://go.dev)

[English](README.md) | 简体中文

---

## 功能特色

- 🖥️ **现代终端 UI** — 使用 [tview](https://github.com/rivo/tview) 构建美观的终端界面
- 🌐 **代理管理** — 实时浏览和切换代理节点与代理组
- ⚙️ **配置控制** — 一键切换 TUN 模式、Allow LAN 和代理模式
- 📊 **实时监控** — 实时流量统计（上行/下行速度）、内存使用和活跃连接数
- 📋 **规则管理** — 查看、添加、编辑、删除、调换顺序和批量导入代理规则与 RULE-SET 规则
- 🔖 **规则提供者** — 添加、更新和删除规则提供者订阅（HTTP/本地文件）
- 📥 **订阅管理** — 从 URL 或本地文件导入代理订阅，合并到现有配置
- 📦 **代理组管理** — 创建、更新和删除自定义代理组，支持 `select`、`url-test`、`fallback`、`load-balance` 类型
- 📝 **实时日志** — 实时日志流式展示，支持等级过滤和暂停/继续
- 🔗 **连接监控** — 查看活跃连接，关闭单个或全部连接，自动刷新
- 🖱️ **完整鼠标支持** — 点击导航、切换标签页、切换按钮
- ⌨️ **键盘快捷键** — F 键、Ctrl+数字、Alt+字母快速切换页面
- 📎 **智能配置合并** — 将订阅配置与本地配置智能合并（代理、组、规则），支持冲突解决

> 👉 [**QUICKSTART_ZH.md**](QUICKSTART_ZH.md) — 安装、配置、首次使用完整指南

## 项目状态

当前版本：**`v0.0-Alpha`** — 开发中

### 开发进度

- [x] API 客户端开发
- [x] UI 框架搭建与布局
- [x] 核心功能实现
- [x] 代理管理（查看 & 切换）
- [x] 实时流量监控
- [x] 实时日志流式展示
- [x] 连接监控
- [x] 仪表盘（模式/配置控制）
- [x] 应用配置（API 地址 & Secret）
- [x] 规则管理（查看、添加、编辑、删除、批量导入）
- [x] 规则提供者管理
- [x] 订阅导入与配置合并
- [x] 代理组管理（CRUD）
- [ ] 查看配置详情
- [ ] 切换配置文件
- [ ] 从界面修改端口
- [ ] 多语言支持（i18n）

## 技术栈

| 层面        | 技术                                              |
|------------|---------------------------------------------------|
| 语言       | Go 1.24.1                                         |
| UI 框架    | [tview](https://github.com/rivo/tview)            |
| 终端       | [tcell](https://github.com/gdamore/tcell/v2)      |
| 配置格式   | JSON（应用配置）+ YAML（mihomo 配置）              |
| API        | Mihomo REST API（HTTP）                           |
| 流式数据   | Mihomo SSE/chunked 端点（流量、日志、内存）        |

## 项目结构

```
mihomoTui/
├── main.go                          # 程序入口
├── go.mod / go.sum                  # Go 模块定义
├── LICENSE                          # MIT 许可证
├── .gitignore
├── static/
│   └── image.png                    # 演示截图
│
├── internal/
│   ├── app.go                       # 应用编排、页面绑定、布局
│   ├── hotkey.go                    # 全局键盘快捷键处理
│   │
│   ├── api/
│   │   ├── client.go                # HTTP API 客户端（所有 Mihomo 端点）
│   │   └── stream.go                # SSE 流式客户端（流量、日志、内存）
│   │
│   ├── cache/
│   │   ├── cache.go                 # 内存 TTL 缓存（线程安全）
│   │   └── cache_test.go
│   │
│   ├── config/
│   │   ├── config.go                # 应用配置管理器（JSON、保存/加载/备份）
│   │   └── mihomo_config.go         # Mihomo YAML 配置 CRUD（组、规则、提供者）
│   │
│   ├── event/
│   │   ├── bus.go                   # 发布/订阅事件总线
│   │   └── bus_test.go
│   │
│   ├── merge/
│   │   ├── engine.go                # 智能配置合并引擎
│   │   └── engine_test.go
│   │
│   ├── models/
│   │   └── clash.go                 # 数据模型：Config、Proxy、Connection、Rule 等
│   │
│   ├── parser/
│   │   ├── parser.go                # YAML 配置解析器（Clash/Mihomo 格式）
│   │   ├── parser_test.go
│   │   ├── util.go                  # 解析工具
│   │   └── validate.go              # 验证工具
│   │
│   ├── proxygroup/
│   │   ├── manager.go               # 代理组管理与依赖分析
│   │   └── manager_test.go
│   │
│   ├── subscription/
│   │   └── manager.go               # 订阅导入（URL 或本地文件）
│   │
│   ├── types/
│   │   ├── config.go                # ConfigDocument — 规范化解析配置
│   │   ├── event.go                 # 事件类型（订阅、配置、代理、缓存）
│   │   ├── merge.go                 # 合并策略与结果类型
│   │   ├── misc.go                  # UpdateTask、CacheEntry、ReloadEvent
│   │   ├── proxygroup.go            # ProxyGroupDef、ImportResult、ConflictEntry
│   │   └── subscription.go          # Subscription & UserInfo
│   │
│   ├── ui/
│   │   ├── ui.go                    # UI 更新器单例（QueueUpdate 封装）
│   │   ├── theme.go                 # 主题颜色和全局样式初始化
│   │   │
│   │   ├── components/
│   │   │   ├── header.go            # 顶部标题栏（应用名、版本、连接状态）
│   │   │   ├── navbar.go            # 水平导航栏（11 个页面）
│   │   │   ├── statusbar.go         # 底部状态栏（模式、流量、操作提示）
│   │   │   └── navigation.go        # 焦点导航原语
│   │   │
│   │   └── pages/
│   │       ├── pages.go             # 页面构造函数 & ActivatablePage 接口
│   │       ├── dashboard.go         # 仪表盘：连接统计、系统信息、TUN/AllowLAN 控制
│   │       ├── proxies.go           # 代理页面：提供者列表 → 组列表 → 节点表格 + 延迟测试
│   │       ├── connections.go       # 连接页面：实时连接表格、搜索、关闭连接
│   │       ├── config.go            # 配置页面：API 地址 & Secret 表单
│   │       ├── logs.go              # 日志页面：实时流式展示，支持暂停/继续
│   │       ├── rules.go             # 规则页面：查看/添加/编辑/删除/批量导入规则
│   │       ├── settings.go          # 设置页面（开发中）
│   │       ├── proxygroups/         # 代理组管理：CRUD、类型选择
│   │       ├── ruleproviders/       # 规则提供者管理：添加/更新/删除
│   │       └── subscriptions/       # 订阅管理：导入、合并预览
│   │
│   └── utils/
│       ├── convert.go               # FormatBytes — 人类可读字节格式化
│       └── env.go                   # 环境变量助手函数
```

## 页面概览

| # | 页面       | 说明 |
|---|-----------|------|
| 1 | **仪表盘** | 连接统计、系统内存、代理模式、TUN/AllowLAN 切换按钮 |
| 2 | **代理**   | 提供者列表 → 组列表 → 节点表格，支持延迟测试 |
| 3 | **连接**   | 实时连接表格，支持搜索、关闭单个或全部连接 |
| 4 | **配置**   | Mihomo API 地址和 Secret 配置 |
| 5 | **日志**   | 实时日志流式展示，支持暂停/继续和等级过滤 |
| 6 | **订阅**   | 从 URL 或文件导入代理订阅，合并到配置 |
| 7 | **代理组** | 创建、更新、删除自定义代理组 |
| 8 | **规则**   | 查看、添加、编辑、删除、调换顺序（↑/↓/移动至）、批量导入代理规则 |
| 9 | **规则提供者** | 添加、更新、删除规则提供者订阅 |

## 键盘快捷键

| 键               | 动作                |
|------------------|--------------------|
| `F1` / `Ctrl+1` / `Alt+D` | 仪表盘   |
| `F2` / `Ctrl+2` / `Alt+P` | 代理     |
| `F3` / `Ctrl+3` / `Alt+R` | 连接     |
| `F4` / `Ctrl+4` / `Alt+C` | 配置     |
| `F5` / `Ctrl+5` / `Alt+L` | 日志     |
| `F6` / `Ctrl+6` / `Alt+S` | 订阅     |
| `F7` / `Ctrl+7` / `Alt+G` | 代理组   |
| `F8` / `Ctrl+8` / `Alt+U` | 规则     |
| `F9` / `Ctrl+9` / `Alt+T` | 规则提供者 |
| `F10` / `Ctrl+0` / `Alt+K`| 设置     |
| `F11` / `Alt+M`           | 配置管理 |
| `Ctrl+C` / `Ctrl+Q`       | 退出程序 |
| `Esc`            | 返回焦点到导航栏 |
| `Tab` / `Shift+Tab` | 循环切换控件焦点 |
| `Ctrl+R`         | 刷新当前页面数据 |
| `?`             | 显示/隐藏快捷键帮助 |

### 规则页面快捷键

| 键               | 动作                |
|------------------|--------------------|
| `A`             | 添加规则             |
| `D`             | 删除选中规则         |
| `R`             | 刷新规则             |
| `P`             | 粘贴导入规则         |
| `M`             | 移动选中规则至指定位置 |
| `↑`/`↓`        | 导航规则列表         |
| `Alt+↑`         | 上移选中规则         |
| `Alt+↓`         | 下移选中规则         |
| `/`             | 聚焦搜索过滤框       |

> 也完全支持鼠标导航 — 点击导航栏、按钮和表格单元格即可操作。

## 快速开始

### 前置条件

- Go 1.24.1 或更高版本
- 支持 256 色和鼠标的终端
- 正在运行的 [Mihomo](https://github.com/MetaCubeX/mihomo) 实例（已启用 external-controller）

### 安装与运行

```bash
# 克隆仓库
git clone <your-repo-url>/mihomoTui.git
cd mihomoTui

# 安装依赖
go mod tidy

# 运行程序
go run main.go
```

### 首次使用

1. 启动应用程序
2. 导航到 **配置** 页面（点击导航栏或按 `F4`/`Ctrl+4`）
3. 输入你的 Mihomo API 地址（默认：`http://127.0.0.1:9090`）
4. 输入 API Secret（如果在 Mihomo 中配置了）
5. 保存 — 配置文件将写入 `~/.config/mihomoTui/config.json`
6. **重启应用程序** 以使设置生效

底部状态栏会显示当前代理模式和实时流量，连接成功后可看到。

## 配置文件

### 应用配置 (`~/.config/mihomoTui/config.json`)

```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  }
}
```

### Mihomo 配置

TUI 可以直接修改运行中的 Mihomo config.yaml，它通过以下方式定位配置文件：
1. 优先使用显式设置的 `MIHOMO_CONFIG_PATH`
2. 扫描 `/proc` 查找运行中的 mihomo 进程及其 `-d` / `--directory` 参数
3. 回退到常见位置：`/etc/mihomo/config.yaml`、`~/.config/mihomo/config.yaml`

对 mihomo 配置的操作（代理组、规则、提供者）需要 **sudo** 权限才能写入配置文件（通常由 root 所有）。

## 开发

### 从源码构建

```bash
# 构建当前平台
go build -o mihomoTui .

# 交叉编译（Linux amd64）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-linux-amd64 .

# 交叉编译（macOS ARM64）
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-darwin-arm64 .

# 交叉编译（Windows）
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-windows-amd64.exe .
```

### 运行测试

```bash
go test ./internal/...
```

## 参与贡献

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 本项目
2. 创建你的功能分支：`git checkout -b feature/AmazingFeature`
3. 提交更改：`git commit -m 'Add some AmazingFeature'`
4. 推送到分支：`git push origin feature/AmazingFeature`
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证，详情请参阅 [LICENSE](LICENSE) 文件。

## 鸣谢

- [Mihomo](https://github.com/MetaCubeX/mihomo) — 核心代理引擎
- [tview](https://github.com/rivo/tview) — 强大的 Go 终端 UI 库
- [tcell](https://github.com/gdamore/tcell/v2) — 底层终端处理
