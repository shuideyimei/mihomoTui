package pages

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const helpText = `[yellow::b]=== 快捷键帮助 ===[-::-]

[green::b]页面导航:[-::-]
  [white]F1 / Ctrl+1 / Alt+D[-]  仪表板
  [white]F2 / Ctrl+2 / Alt+P[-]  代理
  [white]F3 / Ctrl+3 / Alt+R[-]  连接
  [white]F4 / Ctrl+4 / Alt+C[-]  配置
  [white]F5 / Ctrl+5 / Alt+L[-]  日志
  [white]F6 / Ctrl+6 / Alt+S[-]  订阅
  [white]F7 / Ctrl+7 / Alt+G[-]  代理组
  [white]F8 / Ctrl+8 / Alt+U[-]  规则
  [white]F9 / Ctrl+9 / Alt+T[-]  规则提供者
  [white]F10 / Ctrl+0 / Alt+K[-] 设置

[green::b]全局操作:[-::-]
  [white]Esc[-]       返回侧边栏
  [white]Tab[-]       切换焦点
  [white]Ctrl+R[-]    刷新当前页面
  [white]Ctrl+C/Q[-]  退出程序
  [white]?[-]         显示/隐藏帮助

[green::b]页面内操作:[-::-]
  [white]代理页:[-] 空格=延迟测试, R=单节点测速
  [white]连接页:[-] D/Del=关闭连接, F5=刷新, T=自动刷新
  [white]规则页:[-] /=搜索, A=添加, D=删除

[yellow]按 ? 或 Esc 关闭此面板[-]`

// NewHelpPage creates a help overlay panel showing all keyboard shortcuts
func NewHelpPage() tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(helpText)

	textView.SetBorder(true).
		SetTitle(" 帮助 ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorYellow)

	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(textView, 28, 0, true).
		AddItem(nil, 0, 1, false)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 70, 0, true).
		AddItem(nil, 0, 1, false)
}
