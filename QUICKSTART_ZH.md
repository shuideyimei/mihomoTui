# 快速开始指南 (Quick Start Guide)

本指南将帮助你从零开始安装、配置并使用 **mihomoTui**。

---

## 📋 前置条件

1. **Go 1.24+**（用于从源码编译安装）
2. **支持 256 色 / UTF-8 与鼠标的现代终端**（如 iTerm2, WezTerm, Alacritty, GNOME Terminal, Windows Terminal 等）
3. **正在运行的 [Mihomo](https://github.com/MetaCubeX/mihomo)** 代理核心，且已开启 `external-controller` API 接口。

### 检查终端环境

```bash
# 检查终端颜色数（输出应 >= 256）
echo $TERM
tput colors

# 确认已设置 UTF-8 语言环境
echo $LANG
```

---

## 📥 安装与运行

### 1. 克隆源码仓库

```bash
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui
```

### 2. 下载依赖并运行

```bash
go mod tidy
go run main.go
```

### 3. 本地编译二进制文件

```bash
# 编译生成可执行文件
go build -o mihomoTui .

# 启动运行
./mihomoTui
```

---

## ⚡ 首次使用与快速配置

### 步骤 1：启动应用并进入设置

首次启动 mihomoTui 时：
1. 按 `F8`（或 `Alt+8` / 点击左侧导航栏的 **设置**）进入系统设置页面。
2. 在 **API 地址** 输入框中填写 Mihomo 外部控制器地址（默认：`http://127.0.0.1:9090`，或 Unix Socket 地址如 `unix:///tmp/mihomo.sock`）。
3. 如果 Mihomo 配置了 `secret` 密钥，请在 **API 密钥** 中输入对应密钥。
4. 点击 **保存** 按钮，设置会自动持久化到 `~/.config/mihomoTui/config.json`。

### 步骤 2：验证连接与模式切换

配置保存后，顶部标题栏将变为绿色连接状态 `● 已连接` 并显示 Mihomo 核心版本号；底部状态栏将展示实时上/下行速率与当前模式。
- 在任意界面按 `F1` 回到 **仪表盘** 监控流量与内存。
- 按 `F2` 进入 **代理** 页面，按 `T` 键可对当前节点或分组进行批量 HTTP Ping 延迟测速；按 `R`（Rule）、`G`（Global）、`D`（Direct）即可一键切换运行模式。

---

## 🗺️ 页面与键盘快捷键速查

| 快捷键 | 页面 / 功能 | 核心操作说明 |
|:---|:---|:---|
| `F1` / `Alt+1` | **仪表盘 (Dashboard)** | 监控上下行流量、系统内存、连接数，一键切换 TUN 模式与 Allow LAN |
| `F2` / `Alt+2` | **代理 (Proxies)** | 浏览代理组与节点表格，按 `T` 测速，按 `/` 模糊搜索过滤 |
| `F3` / `Alt+3` | **连接 (Connections)** | 监控活跃连接，按 `D` 阻断单条连接，按 `C` 关闭全部连接，按 `Space` 暂停/恢复 |
| `F4` / `Alt+4` | **日志 (Logs)** | 实时 SSE 日志流展示，按级别过滤（Debug/Info/Warn/Error），按 `Space` 暂停 |
| `F5` / `Alt+5` | **配置文件 (Profiles)** | 发现并切换运行配置、新建配置，内置在线 YAML 编辑器（带防丢退出保护） |
| `F6` / `Alt+6` | **订阅管理 (Subscriptions)**| 输入订阅链接或本地文件一键导入，智能解析节点与剩余流量信息 |
| `F7` / `Alt+7` | **配置编辑器 (Editor)** | 聚合配置编辑器，按 `Tab` 在 **代理组**、**路由规则**、**规则提供者** 间切换 |
| `F8` / `Alt+8` | **系统设置 (Settings)** | 修改 API Endpoint / Secret，一键切换中文/英文（无需重启即时刷新） |
| `:` | **命令面板 (Command Palette)**| 唤起快速搜索输入框，模糊匹配直达任意功能与页面 |
| `?` / `F12` | **帮助中心 (Help)** | 唤起全局交互快捷键指南弹窗 |
| `Esc` | **返回导航** | 退出输入状态，将焦点返回到左侧导航栏 |
| `Tab` / `Shift+Tab`| **焦点切换** | 在当前页面的表单、表格与按钮之间循环切换焦点 |

---

## ⚙️ 配置文件说明

### 1. mihomoTui 应用配置
文件路径：`~/.config/mihomoTui/config.json`
```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  },
  "language": "zh"
}
```

### 2. Mihomo 核心配置定位
mihomoTui 会自动通过以下层级寻找正在运行的 Mihomo `config.yaml`：
1. 环境变量 `MIHOMO_CONFIG_PATH` 指定的路径；
2. 解析当前运行中的 Mihomo 进程命令行参数（`-d`, `-f`, `--config`）；
3. 系统常见默认路径（`~/.config/mihomo/config.yaml`, `/etc/mihomo/config.yaml`, `~/.config/clash/config.yaml`）。

---

## 🛠️ 跨平台编译打包

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-linux-amd64 .

# Linux arm64 (Raspberry Pi / 嵌入式开发板)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-linux-arm64 .

# macOS ARM64 (Apple Silicon M1/M2/M3/M4)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mihomoTui-darwin-arm64 .

# Windows 64位
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o mihomoTui-windows-amd64.exe .
```

---

## ❓ 常见问题排查 (Troubleshooting)

### 1. 顶部标题栏显示「未连接」
- 检查 Mihomo 核心是否已启动并开启外部控制器：在终端运行 `curl http://127.0.0.1:9090/version` 测试返回。
- 确认 `F8` 设置页面中填写的 API 地址与 Secret 是否与 Mihomo `config.yaml` 中的 `external-controller` 及 `secret` 完全一致。

### 2. 编辑配置提示权限不足 (Permission Denied)
- 若系统 Mihomo 运行在 root 权限下（如 `/etc/mihomo/config.yaml` 归 root 所有），可以通过 `sudo ./mihomoTui` 运行，或将配置文件所有权授予当前用户：
  ```bash
  sudo chown $USER ~/.config/mihomo/config.yaml
  ```
