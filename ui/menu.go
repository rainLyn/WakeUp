/*
 * ui/menu.go — 主菜单页面
 *
 * 职责：
 *   1. 展示功能入口，j/k 导航，1-6 数字快捷键
 *   2. 大部分选项直达设备列表（列表页是主操作界面，内嵌唤醒/编辑/删除）
 *
 * 导航规则：
 *   "新增设备" → 直接进入表单页
 *   "退出程序" → tea.Quit
 *   其余选项 → 统一进入设备列表页（带上下文提示）
 */

package ui

import (
	"fmt"
	"strings"

	"WakeUp/store"

	tea "github.com/charmbracelet/bubbletea"
)

/* ---------------------------------------------------------------------------
 * 菜单选项
 * ---------------------------------------------------------------------------*/

type menuItem struct {
	number      int
	label       string
	description string
}

var menuItems = []menuItem{
	{1, "查看设备列表", "浏览所有已保存的唤醒设备"},
	{2, "新增唤醒设备", "添加一台新的网络唤醒设备"},
	{3, "编辑设备信息", "修改已有设备的参数（名称/MAC/端口/地址）"},
	{4, "删除设备", "移除不再需要的设备"},
	{5, "唤醒设备", "选择设备发送 WOL 网络唤醒包"},
	{6, "退出程序", "关闭 WakeUp"},
}

/* ---------------------------------------------------------------------------
 * MenuModel
 * ---------------------------------------------------------------------------*/

// MenuModel 主菜单页面模型
type MenuModel struct {
	store  *store.Store
	cursor int
	width  int
	height int
}

// NewMenuModel 创建主菜单模型
func NewMenuModel(s *store.Store, w, h int) MenuModel {
	return MenuModel{
		store:  s,
		cursor: 0,
		width:  w,
		height: h,
	}
}

/* ---------------------------------------------------------------------------
 * Bubble Tea 接口
 * ---------------------------------------------------------------------------*/

func (m MenuModel) Init() tea.Cmd { return nil }

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {

		case "j", "down":
			m.cursor = min(m.cursor+1, len(menuItems)-1)
		case "k", "up":
			m.cursor = max(m.cursor-1, 0)

		case "1":
			return m.goToList("")
		case "2":
			return NewFormModel(m.store, m.width, m.height, -1), nil
		case "3":
			return m.goToList("选择一个设备，按 e 进入编辑")
		case "4":
			return m.goToList("选择一个设备，按 dd 删除")
		case "5":
			return m.goToList("选择一个设备，按 Enter 发送唤醒包")
		case "6":
			return m, tea.Quit

		case "enter", " ":
			return m.handleSelect()

		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder

	b.WriteString(PadTop(2))
	b.WriteString(RenderTitle("🌅 WakeUp · 网络唤醒工具", m.width))
	b.WriteString("\n\n")

	for i, item := range menuItems {
		prefix := Arrow(i == m.cursor)
		line := fmt.Sprintf("%d. %s  %s",
			item.number, item.label,
			HintStyle.Render(item.description))

		if i == m.cursor {
			b.WriteString(SelectedStyle.Render(prefix + line))
		} else {
			b.WriteString(NormalRowStyle.Render(prefix + line))
		}
		b.WriteString("\n")
	}

	// 帮助栏固定在底部
	helpBar := RenderHelpBar(m.width, []KeyBinding{
		{"j/k", "上下移动"}, {"1-6", "快速选择"},
		{"Enter", "确认"}, {"q", "退出"},
	})
	return PinBottom(m.height, b.String(), helpBar)
}

/* ---------------------------------------------------------------------------
 * 页面路由
 * ---------------------------------------------------------------------------*/

func (m MenuModel) goToList(hint string) (tea.Model, tea.Cmd) {
	l := NewListModel(m.store, m.width, m.height)
	l.hint = hint
	return l, nil
}

func (m MenuModel) handleSelect() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // 查看设备
		return m.goToList("")
	case 1: // 新增设备
		return NewFormModel(m.store, m.width, m.height, -1), nil
	case 2: // 编辑设备
		return m.goToList("选择一个设备，按 e 进入编辑")
	case 3: // 删除设备
		return m.goToList("选择一个设备，按 dd 删除")
	case 4: // 唤醒设备
		return m.goToList("选择一个设备，按 Enter 发送唤醒包")
	case 5: // 退出
		return m, tea.Quit
	default:
		return m, nil
	}
}
