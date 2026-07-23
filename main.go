/*
 * main.go — WakeUp 网络唤醒工具入口
 *
 * 职责：
 *   1. 初始化数据存储（~/.wakeup/devices.json）
 *   2. 创建顶层 Bubble Tea 模型
 *   3. 启动 TUI 程序
 *
 * 架构：
 *   顶层 Model 只做三件事：
 *     1. 捕获全局快捷键（ctrl+c 退出）
 *     2. 捕获全局异步消息（SaveResultMsg 静默处理）
 *     3. 将其他一切委托给当前页面 Model
 *
 *   页面切换由各页面自身决定——Update() 返回的就是下一页。
 *   这消除了顶层 switch pageType 的分支逻辑。
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"WakeUp/store"
	"WakeUp/ui"

	tea "github.com/charmbracelet/bubbletea"
)

/* ---------------------------------------------------------------------------
 * 顶层模型
 * ---------------------------------------------------------------------------*/

// model 是程序的顶层 Bubble Tea 模型
type model struct {
	page   tea.Model
	store  *store.Store
	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return m.page.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ---- 全局窗口尺寸 ----
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	// ---- 全局退出快捷键（任何页面都有效） ----
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// ---- 全局异步消息：保存结果（静默消费，不透传给页面） ----
	if _, ok := msg.(ui.SaveResultMsg); ok {
		return m, nil
	}

	// ---- 委托给当前页面 ----
	newPage, cmd := m.page.Update(msg)
	m.page = newPage
	return m, cmd
}

func (m model) View() string {
	return m.page.View()
}

/* ---------------------------------------------------------------------------
 * 程序入口
 * ---------------------------------------------------------------------------*/

func main() {
	// 数据目录：~/.wakeup/
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
		os.Exit(1)
	}
	dataDir := filepath.Join(home, ".wakeup")

	s, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据存储失败: %v\n", err)
		os.Exit(1)
	}

	// 初始页面：设备列表（一级页面，直接可唤醒）
	m := model{
		page:   ui.NewListModel(s, 80, 24),
		store:  s,
		width:  80,
		height: 24,
	}

	// 启用 Alt Screen 缓冲
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "程序运行异常: %v\n", err)
		os.Exit(1)
	}
}
