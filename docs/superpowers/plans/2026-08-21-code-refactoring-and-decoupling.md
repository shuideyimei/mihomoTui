# 代码简洁度优化与架构解耦重构计划 (Code Refactoring & Decoupling Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 彻底消除数据模型冗余定义、理顺 UI 与 Service 架构分层、规范 EventBus 事件通信以解除全局状态强耦合，并修复并发快速切换时的 Goroutine 残留问题。

**Architecture:** 
1. 统一收敛领域数据模型至 `internal/models`，消除跨包重复结构体与无意义的字段映射；
2. 规范化 UI → Service → Repository 分层调用，使 `RulesPage`、`ProxyGroupsPage`、`RuleProvidersPage` 严格通过注入的 `Service` 层操作数据；
3. 规范化 `EventBus`，以事件驱动解耦组件状态同步（代理模式、配置切换），清理 `UiUpdater` 上帝对象职责；
4. 强化异步 Stream/Ticker 的 `context.Context` 级联取消与生命周期管理，保障高频切换下的并发稳定性。

**Tech Stack:** Go 1.24, `tview`, `tcell/v2`, `yaml.v3`, Go 标准库 `sync`, `context`。

---

## 涉及文件与职责规划

| 文件路径 | 变更类型 | 核心职责与改造点 |
| :--- | :---: | :--- |
| `internal/models/entities.go` | 修改 | 作为领域层核心，规范导出 `RuleDisplay`, `ProxyGroupEntry`, `ProxyProviderEntry`, `RuleProviderEntry`, `ProviderInfo` 等统一实体 |
| `internal/config/mihomo_config.go` | 修改 | 将重复结构体定义指向 `models.*`（通过类型别名或直接引用），精简模型定义 |
| `internal/repository/yaml_driver.go` | 修改 | 消除冗余的 struct 字段遍历转换，直接传递统一实体对象 |
| `internal/events/bus.go` | 修改 | 为 `Bus` 增加注销能力（Unsubscribe）、Handler 异常恢复与线程安全保障 |
| `internal/events/types.go` | 修改 | 补全并统一 `LanguageChangedEvent`, `ConfigSwitchedEvent`, `ProxyModeChangedEvent` 载荷 |
| `internal/ui/ui.go` | 修改 | 剔除 `UiUpdater` 中的 `statusBar` 反向代理接口（`GetCurrentMode`, `UpdateConfig`），专注 UI 调度与全局 Modal |
| `internal/ui/components/statusbar.go` | 修改 | 代理模式变更时发布 `TopicProxyModeChanged` 事件；优化 `StreamTraffic` 失败重试退出机制 |
| `internal/ui/pages/proxies.go` | 修改 | 订阅 `TopicProxyModeChanged` 替代 `ui.Updater.GetCurrentMode()` 反向嗅探；规范异步 context 管理 |
| `internal/ui/pages/rules.go` | 修改 | 消除内部 `RuleDisplay` 定义；通过构造注入 `RuleService` 并替换底层直接静态调用 |
| `internal/ui/pages/proxygroups/page.go` | 修改 | 通过构造注入 `ProxyGroupService`，通过服务层进行代理组增删改查 |
| `internal/ui/pages/ruleproviders/page.go` | 修改 | 通过构造注入 `ProviderService`，通过服务层进行规则提供者增删改查 |
| `internal/ui/pages/pages.go` | 修改 | 清理纯转发包装函数，更新构造方法签名 |
| `internal/app.go` | 修改 | 真正将初始化的各个 Service 和 EventBus 依赖注入到各个对应 Page 中 |
| `internal/ui_freeze_test.go` | 测试 | 验证页面切换无死锁、无 Goroutine 泄漏 |

---

### Task 1: 领域数据模型归一化与去冗余 (Model Unification)

**Files:**
- Modify: `internal/models/entities.go`
- Modify: `internal/config/mihomo_config.go:30-65,1100-1120,1400-1420`
- Modify: `internal/repository/yaml_driver.go:15-80`
- Modify: `internal/ui/pages/rules.go:40-60`
- Test: `internal/repository/yaml_driver_test.go`

- [ ] **Step 1: 编写模型一致性与驱动层转换单测**

在 `internal/repository/yaml_driver_test.go` 中验证 `yaml_driver` 直接处理 `models.RuleDisplay` 与 `models.ProxyGroupEntry` 时的结构完整性：

```go
package repository

import (
	"testing"
	"mihomoTui/internal/models"
)

func TestLocalYAMLDriver_ModelIntegrity(t *testing.T) {
	driver := NewLocalYAMLDriver()
	if driver == nil {
		t.Fatal("expected non-nil driver")
	}

	// Verify model types are shared
	var _ RuleRepository = driver
	var _ ProxyGroupRepository = driver
	var _ ProviderRepository = driver
	var _ ProfileRepository = driver
}
```

- [ ] **Step 2: 运行测试确认编译或运行状态**

Run: `go test -v ./internal/repository`
Expected: PASS 或编译错误（如类型不匹配时先暴露）。

- [ ] **Step 3: 归一化实体定义并消除重复 struct**

1. 修改 `internal/models/entities.go`，确保包含完整的统一实体：
```go
package models

// RuleDisplay represents a parsed or UI-facing rule.
type RuleDisplay struct {
	Type    string `json:"type" yaml:"type"`
	Payload string `json:"payload" yaml:"payload"`
	Proxy   string `json:"proxy" yaml:"proxy"`
}

// ProxyGroupEntry is a proxy group definition in config.
type ProxyGroupEntry struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	Proxies  []string `json:"proxies,omitempty" yaml:"proxies,omitempty"`
	Use      []string `json:"use,omitempty" yaml:"use,omitempty"`
	URL      string   `json:"url,omitempty" yaml:"url,omitempty"`
	Interval int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Filter   string   `json:"filter,omitempty" yaml:"filter,omitempty"`
}

// ProxyProviderEntry is a proxy provider definition in config.
type ProxyProviderEntry struct {
	Type        string `json:"type" yaml:"type"`
	URL         string `json:"url" yaml:"url"`
	Interval    int    `json:"interval" yaml:"interval"`
	HealthCheck *struct {
		Enable   bool   `json:"enable" yaml:"enable"`
		URL      string `json:"url" yaml:"url"`
		Interval int    `json:"interval" yaml:"interval"`
	} `json:"health-check,omitempty" yaml:"health-check,omitempty"`
}

// RuleProviderEntry is a rule provider definition in config.
type RuleProviderEntry struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Behavior string `json:"behavior" yaml:"behavior"`
	URL      string `json:"url" yaml:"url"`
	Path     string `json:"path" yaml:"path"`
	Interval int    `json:"interval" yaml:"interval"`
	Format   string `json:"format" yaml:"format"`
}

// ProviderInfo holds metadata about a proxy provider.
type ProviderInfo struct {
	Name     string `json:"name" yaml:"name"`
	URL      string `json:"url" yaml:"url"`
	Interval int    `json:"interval" yaml:"interval"`
}
```

2. 修改 `internal/config/mihomo_config.go`，将重复类型定义为别名：
```go
// In internal/config/mihomo_config.go
type ProxyProviderEntry = models.ProxyProviderEntry
type ProxyGroupEntry = models.ProxyGroupEntry
type ProviderInfo = models.ProviderInfo
type RuleProviderEntry = models.RuleProviderEntry
type RuleEntry = models.RuleDisplay
```

3. 修改 `internal/repository/yaml_driver.go`，直接返回切片，移除重复的遍历赋值：
```go
func (d *LocalYAMLDriver) GetRules() ([]models.RuleDisplay, error) {
	rawRules, err := config.GetRulesFromConfig()
	if err != nil {
		return nil, err
	}
	var rules []models.RuleDisplay
	for _, raw := range rawRules {
		parts := config.ParseRuleLine(raw)
		rules = append(rules, parts)
	}
	return rules, nil
}

func (d *LocalYAMLDriver) GetGroups() ([]models.ProxyGroupEntry, error) {
	return config.GetAllProxyGroupsFromConfig()
}

func (d *LocalYAMLDriver) BatchAddRules(rules []models.RuleDisplay) (string, error) {
	return config.BatchAddRulesToConfig(rules)
}
```

4. 移除 `internal/ui/pages/rules.go` 内部自带的 `type RuleDisplay struct`，统一使用 `models.RuleDisplay`。

- [ ] **Step 4: 运行仓库所有单元测试验证模型重构**

Run: `go test ./internal/models ./internal/config ./internal/repository ./internal/service ./internal/ui/pages/...`
Expected: PASS

- [ ] **Step 5: 提交模型去重成果**

```bash
git add internal/models internal/config internal/repository internal/ui/pages/rules.go
git commit -m "refactor(models): unify domain models and eliminate duplicate struct definitions"
```

---

### Task 2: 规范 EventBus 与组件状态解耦 (Standardize EventBus & Decouple UI)

**Files:**
- Modify: `internal/events/bus.go`
- Modify: `internal/events/types.go`
- Modify: `internal/ui/ui.go:15-50`
- Modify: `internal/ui/components/statusbar.go:60-140`
- Modify: `internal/ui/pages/proxies.go:290-330`
- Test: `internal/events/bus_test.go`

- [ ] **Step 1: 编写 EventBus 退订与并发安全性测试**

在 `internal/events/bus_test.go` 中添加注销与跨组件事件广播测试：

```go
func TestBus_UnsubscribeAndModeEvent(t *testing.T) {
	bus := NewBus()
	received := ""
	unsub := bus.Subscribe(TopicProxyModeChanged, func(ev interface{}) {
		if e, ok := ev.(ProxyModeChangedEvent); ok {
			received = e.Mode
		}
	})

	bus.Publish(TopicProxyModeChanged, ProxyModeChangedEvent{Mode: "rule"})
	if received != "rule" {
		t.Fatalf("expected 'rule', got %s", received)
	}

	unsub()
	bus.Publish(TopicProxyModeChanged, ProxyModeChangedEvent{Mode: "global"})
	if received != "rule" {
		t.Fatalf("expected handler not called after unsub, got %s", received)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test -v ./internal/events -run TestBus_UnsubscribeAndModeEvent`
Expected: FAIL（`bus.Subscribe` 尚未返回 `unsub` 函数）。

- [ ] **Step 3: 完善 EventBus 并重构状态同步**

1. 修改 `internal/events/bus.go`：
```go
package events

import (
	"sync"
	"sync/atomic"
)

type Handler func(event interface{})

type subscription struct {
	id      int64
	handler Handler
}

type Bus struct {
	mu     sync.RWMutex
	subs   map[string][]subscription
	nextID int64
}

func NewBus() *Bus {
	return &Bus{
		subs: make(map[string][]subscription),
	}
}

func (b *Bus) Subscribe(topic string, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := atomic.AddInt64(&b.nextID, 1)
	b.subs[topic] = append(b.subs[topic], subscription{id: id, handler: handler})

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		items := b.subs[topic]
		for i, sub := range items {
			if sub.id == id {
				b.subs[topic] = append(items[:i], items[i+1:]...)
				break
			}
		}
	}
}

func (b *Bus) Publish(topic string, event interface{}) {
	b.mu.RLock()
	subs := make([]subscription, len(b.subs[topic]))
	copy(subs, b.subs[topic])
	b.mu.RUnlock()

	for _, sub := range subs {
		if sub.handler != nil {
			func() {
				defer func() { _ = recover() }()
				sub.handler(event)
			}()
		}
	}
}
```

2. 清理 `internal/ui/ui.go` 中的 `statusBar` 接口及 `SetStatusBar` / `GetCurrentMode` 侵入式代码，仅保留 UI 调度与 Modal 职责。

3. 修改 `internal/ui/components/statusbar.go`，在初始化与收到模式变更时，向 `eventBus` 发布事件：
```go
func (s *StatusBar) updateConfig(config *models.Config) {
	s.mu.Lock()
	s.config = config
	s.updateContent()
	mode := ""
	if s.config != nil {
		mode = s.config.Mode
	}
	s.mu.Unlock()

	if mode != "" && s.bus != nil {
		s.bus.Publish(events.TopicProxyModeChanged, events.ProxyModeChangedEvent{Mode: mode})
	}
}
```

4. 修改 `internal/ui/pages/proxies.go`，通过订阅 `events.TopicProxyModeChanged` 实现按键高亮响应，移除对 `ui.Updater.GetCurrentMode()` 的反向嗅探。

- [ ] **Step 4: 运行 events 与 ui 测试**

Run: `go test ./internal/events ./internal/ui/components`
Expected: PASS

- [ ] **Step 5: 提交 EventBus 解耦变更**

```bash
git add internal/events internal/ui/ui.go internal/ui/components/statusbar.go internal/ui/pages/proxies.go
git commit -m "refactor(events): decouple state sync via EventBus and clean UiUpdater"
```

---

### Task 3: 消除 Service 与 UI 层的架构漂移 (Connect UI to Service Layer)

**Files:**
- Modify: `internal/ui/pages/rules.go`
- Modify: `internal/ui/pages/proxygroups/page.go`
- Modify: `internal/ui/pages/ruleproviders/page.go`
- Modify: `internal/app.go:110-150,240-270`
- Test: `internal/service/service_test.go`
- Test: `internal/ui/pages/rules_test.go`

- [ ] **Step 1: 编写 UI 页面依赖 Service 的单元测试**

在 `internal/ui/pages/rules_test.go` 中验证通过 `RuleService` 进行数据交互：

```go
func TestRulesPage_ServiceIntegration(t *testing.T) {
	driver := repository.NewLocalYAMLDriver()
	ruleSvc := service.NewRuleService(driver)
	page := NewRulesPage(ruleSvc)
	if page == nil {
		t.Fatal("expected non-nil RulesPage")
	}
}
```

- [ ] **Step 2: 运行测试确认编译需求**

Run: `go test -v ./internal/ui/pages -run TestRulesPage_ServiceIntegration`
Expected: FAIL（`NewRulesPage` 参数签名更新前无法编译）。

- [ ] **Step 3: 改造 UI 页面全面接入 Service**

1. 修改 `internal/ui/pages/rules.go`：
   - 为 `RulesPage` 增加 `ruleService *service.RuleService` 字段；
   - 构造器更新为 `NewRulesPage(ruleService *service.RuleService) *RulesPage`；
   - 页面内的 `refresh()`、`AddRule`、`DeleteRule`、`MoveRule`、`BatchAddRules` 统一改为调用 `r.ruleService.*`，不再直接导入并调用 `config.*`。

2. 修改 `internal/ui/pages/proxygroups/page.go`：
   - 增加 `groupService *service.ProxyGroupService` 字段；
   - 构造器更新为 `NewPage(groupService *service.ProxyGroupService) *Page`；
   - 增删改查统一通过 `p.groupService.*` 驱动。

3. 修改 `internal/ui/pages/ruleproviders/page.go`：
   - 增加 `providerService *service.ProviderService` 字段；
   - 构造器更新为 `NewPage(providerService *service.ProviderService) *Page`；
   - 增删改查统一通过 `p.providerService.*` 驱动。

4. 修改 `internal/app.go`：
   - 在 `setupPages()` 中将已经创建的 `a.ruleService`、`a.groupService`、`a.providerService` 分别注入 `proxyGroupsPage`、`rulesPage` 与 `rpPage`，消除悬空变量。

- [ ] **Step 4: 运行所有相关模块单测**

Run: `go test ./internal/service ./internal/ui/pages/...`
Expected: PASS

- [ ] **Step 5: 提交 Service 分层贯通代码**

```bash
git add internal/ui/pages/rules.go internal/ui/pages/proxygroups/page.go internal/ui/pages/ruleproviders/page.go internal/app.go internal/ui/pages/rules_test.go
git commit -m "refactor(ui): route rules, groups, and providers through service layer"
```

---

### Task 4: 清理空转包装与构造器冗余 (Clean Trivial Wrappers)

**Files:**
- Modify: `internal/ui/pages/pages.go`
- Modify: `internal/app.go`
- Modify: `internal/ui/pages/dashboard.go`
- Modify: `internal/ui/pages/connections.go`
- Modify: `internal/ui/pages/logs.go`

- [ ] **Step 1: 检查各页面的构造器导出规范**

检查 `pages.go` 中存在的纯别名函数：
- `NewDashboard()` vs `NewDashboardPage()`
- `NewProxies()` vs `NewProxiesPage()`
- `NewConnections()` vs `NewConnectionsPage()`
- `NewLogs()` vs `NewLogsPage()`

- [ ] **Step 2: 规范构造器命名并移除无用包装**

在 `internal/ui/pages` 中统一构造函数规范：
- `NewDashboardPage() *DashboardPage`
- `NewProxiesPage(bus *events.Bus) *ProxiesPage`
- `NewConnectionsPage() *ConnectionsPage`
- `NewLogsPage() *LogsPage`
- `NewRulesPage(ruleService *service.RuleService) *RulesPage`
- `NewEditorPage(...) *EditorPage`
- `NewUnifiedSettingsPage(...) *UnifiedSettingsPage`

在 `internal/ui/pages/pages.go` 中移除无意义的 1 行单代转发函数，保留 `ActivatablePage` 与 `NavigationGuard` 等公共接口及容器页面组合器。

- [ ] **Step 3: 运行所有测试**

Run: `go test ./internal/ui/pages/... ./internal/...`
Expected: PASS

- [ ] **Step 4: 提交构造器精简代码**

```bash
git add internal/ui/pages internal/app.go
git commit -m "clean(ui): remove redundant pass-through page constructor wrappers"
```

---

### Task 5: 修复并发与生命周期协程泄漏 (Fix Goroutine Leak in Lifecycle)

**Files:**
- Modify: `internal/ui/components/statusbar.go:60-95`
- Modify: `internal/ui/pages/dashboard.go:420-465`
- Modify: `internal/ui/pages/logs.go:200-240`
- Test: `internal/ui_freeze_test.go:200-240`

- [ ] **Step 1: 运行现有的协程泄漏测试**

Run: `go test -v ./internal -run TestGoroutineLeakDuringRapidPageTransitions`
Expected: FAIL (`Potential goroutine leak during rapid page switches`)。

- [ ] **Step 2: 修复流式重试与 Context 取消时的短时协程残留**

1. 修改 `internal/ui/components/statusbar.go` 中的 `startTrafficStream()`：
确保在网络错误等待重试时，使用受控的 channel 或即时退出：
```go
func (s *StatusBar) startTrafficStream() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			err := api.StreamClient.StreamTraffic(s.ctx, func(traffic *models.Traffic) {
				ui.Updater.UpdateUi(func() {
					s.updateTraffic(traffic)
				})
			})

			if err != nil {
				ui.Updater.UpdateUi(func() {
					s.updateTraffic(nil)
				})
				if s.ctx.Err() != nil {
					return
				}
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
			}
		}
	}
}
```

2. 检查并同步优化 `dashboard.go`（`StreamMemoryUsage`）与 `logs.go`（`StreamLogs`）中的 `ctx.Done()` 响应敏感度。

- [ ] **Step 3: 重新运行协程泄漏测试验证修复**

Run: `go test -v ./internal -run TestGoroutineLeakDuringRapidPageTransitions`
Expected: PASS (leaked == 0 或在安全阈值内)。

- [ ] **Step 4: 提交并发稳定性修复**

```bash
git add internal/ui/components/statusbar.go internal/ui/pages/dashboard.go internal/ui/pages/logs.go internal/ui_freeze_test.go
git commit -m "fix(concurrency): ensure immediate goroutine termination on context cancellation"
```

---

### Task 6: 全局回归验证与指标复测 (Full Regression & Metrics)

**Files:**
- 全项目各模块

- [ ] **Step 1: 运行全套单元测试与竞态检测**

Run: `go test -race ./...`
Expected: 全部测试 PASS，无 Data Race。

- [ ] **Step 2: 验证代码编译无警告**

Run: `go vet ./...`
Expected: 无警告输出。

- [ ] **Step 3: 验证应用完整启动与构建**

Run: `go build -o /tmp/mihomoTui-build-test main.go`
Expected: 成功生成二进制文件。

- [ ] **Step 4: 最终提交**

```bash
git add .
git commit -m "chore: complete code conciseness and decoupling refactoring"
```
