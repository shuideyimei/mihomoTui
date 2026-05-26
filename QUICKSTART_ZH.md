# 快速开始

## 前置条件

- **Go 1.24.1** 或更高版本
- 支持 **256 色** 和 **鼠标** 的终端（如 GNOME Terminal、iTerm2、Windows Terminal 等）
- 正在运行的 [Mihomo](https://github.com/MetaCubeX/mihomo) 实例（已启用 `external-controller`）

### 验证你的终端

```bash
# 检查终端颜色数（应输出 ≥ 256）
echo $TERM
tput colors

# 确认终端支持鼠标事件（现代终端默认支持）
```

## 安装

### 1. 克隆仓库

```bash
git clone https://github.com/shuideyimei/mihomoTui.git
cd mihomoTui
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 构建并运行

```bash
# 直接运行（开发模式）
go run main.go

# 或构建为二进制
go build -o mihomoTui .
./mihomoTui
```

## 首次使用

### 步骤 1：启动应用

```bash
go run main.go
```

你应该会看到类似下面的界面（导航栏在顶部，页面在中间，状态栏在底部）：

```
┌─ mihomoTui v0.0-Alpha ────────────────────────── 状态: 已连接 ─┐
│ [Dashboard] [Proxies] [Connections] [Config] [Logs] ...        │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │                     欢迎使用 mihomoTui                        │ │
│ │             请先配置 API 地址和密钥                           │ │
│ └─────────────────────────────────────────────────────────────┘ │
│ 模式: Rule   ↑ 0B/s  ↓ 0B/s  内存: 0MB  连接: 0              │
└─────────────────────────────────────────────────────────────────┘
```

### 步骤 2：配置 API 连接

1. 按 `F4` 或点击导航栏 **Config** 进入配置页面
2. 输入你的 Mihomo API 地址（默认：`http://127.0.0.1:9090`）
3. 输入 API Secret（如果在 Mihomo 中配置了 `secret` 字段）
4. 点击 **保存**（或按 `Enter`）
5. 配置自动写入 `~/.config/mihomoTui/config.json`
6. **重启应用** 使设置生效

### 步骤 3：验证连接

重启后，底部状态栏应显示：
- 当前代理模式（Rule / Global / Direct）
- 实时流量（上行/下行速度）
- 内存使用情况
- 活跃连接数

如果状态栏显示「未连接」，请检查：
- Mihomo 是否正在运行
- API 地址和 Secret 是否正确
- 端口是否可访问

## 配置详解

### 应用配置

配置文件位置：`~/.config/mihomoTui/config.json`

```json
{
  "api": {
    "base_url": "http://127.0.0.1:9090",
    "secret": ""
  }
}
```

| 字段 | 说明 |
|------|------|
| `base_url` | Mihomo API 地址，格式 `http://ip:port` |
| `secret` | API 密钥，与 `config.yaml` 中 `secret` 字段一致 |

### Mihomo 配置要求

你的 Mihomo `config.yaml` 必须启用 `external-controller`：

```yaml
# config.yaml 示例
port: 7890
socks-port: 7891
external-controller: 0.0.0.0:9090  # 必须启用
secret: ""                          # 可选，建议设置
```

### 定位 Mihomo 配置文件

TUI 通过以下方式寻找 Mihomo 的 `config.yaml`：

1. 扫描 `/proc` 查找运行中的 mihomo 进程及其 `-d` / `--directory` 参数
2. 回退到常见位置：
   - `/etc/mihomo/config.yaml`
   - `~/.config/mihomo/config.yaml`

> ⚠️ 对 Mihomo 配置的写操作（代理组、规则、提供者）通常需要 `sudo` 权限，因为配置文件归 root 所有。

## 页面导航

| 键 | 跳转到 |
|----|--------|
| `F1` / `Ctrl+1` / `Alt+D` | 仪表盘 |
| `F2` / `Ctrl+2` / `Alt+P` | 代理 |
| `F3` / `Ctrl+3` / `Alt+R` | 连接 |
| `F4` / `Ctrl+4` / `Alt+C` | 配置 |
| `F5` / `Ctrl+5` / `Alt+L` | 日志 |
| `F6` / `Ctrl+6` / `Alt+S` | 订阅 |
| `F7` / `Ctrl+7` / `Alt+G` | 代理组 |
| `F8` / `Ctrl+8` / `Alt+U` | 规则 |
| `F9` / `Ctrl+9` / `Alt+T` | 规则提供者 |
| `F10` / `Ctrl+0` / `Alt+K` | 设置 |
| `F11` / `Alt+M` | 配置管理 |
| `?` | 显示/隐藏快捷键帮助 |

也支持鼠标点击导航栏切换页面。

## 从源码构建

### 构建当前平台

```bash
go build -o mihomoTui .
```

### 交叉编译

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

### 运行测试

```bash
# 运行所有内部包测试
go test ./internal/...

# 运行特定包测试
go test ./internal/ui/pages/...
go test ./internal/api/...
```

## 疑难解答

### 启动后看不到界面 / 乱码

确保你的终端：
- 支持 UTF-8 编码
- 设置 `TERM=xterm-256color` 或类似值
- 已安装等宽字体

### 状态栏显示「未连接」

1. 确认 Mihomo 正在运行：`ps aux | grep mihomo`
2. 检查 API 是否可访问：`curl http://127.0.0.1:9090/version`
3. 验证配置页面中的地址和密钥是否正确

### 无法保存配置

配置目录 `~/.config/mihomoTui/` 会自动创建。如果保存失败，检查：
- 目录权限是否正确
- 磁盘空间是否充足

### 修改 Mihomo 配置需要 sudo

```bash
# 临时提权运行
sudo ./mihomoTui

# 或修改配置文件权限（不推荐）
sudo chown $USER /etc/mihomo/config.yaml
```
