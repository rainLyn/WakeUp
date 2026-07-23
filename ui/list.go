/*
 * ui/list.go — 设备列表页面（程序主交互界面）
 *
 * 职责：
 *   1. 展示所有已保存设备（表格视图，列均分、居中对齐）
 *   2. Vim 普通模式：j/k 浏览、a 新增、e 编辑、dd 删除、Enter 唤醒
 *   3. 内联弹窗：删除确认、唤醒确认 + 异步发送 + 结果反馈
 *
 * 设计要点：
 *   - Enter 唤醒确认弹窗，Enter/y 均可确认
 *   - dd 双键序列通过 pendingD 标志实现
 *   - 选中行用 ▶ 箭头标识，不用背景色
 *   - 帮助栏通过 PinBottom 始终固定在屏幕底部
 */

package ui

import (
	"fmt"
	"strings"

	"WakeUp/store"
	"WakeUp/wol"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

/* ---------------------------------------------------------------------------
 * ListModel
 * ---------------------------------------------------------------------------*/

// ListModel 设备列表页面模型
type ListModel struct {
	store       *store.Store
	cursor      int    // 当前光标位置
	mode        int    // 当前模态
	pendingD    bool   // d 键前缀等待（dd 双键检测）
	targetIdx   int    // 当前操作目标设备索引（唤醒/删除共用）
	status      string // 内联状态提示（仅表单返回时暂存）
	isError     bool
	resultTitle string // 结果弹窗标题
	resultMsg   string // 结果弹窗内容
	resultIsErr bool   // 结果是否为错误
	hint        string // 页面上下文提示
	width       int
	height      int
}

// NewListModel 创建设备列表模型
func NewListModel(s *store.Store, w, h int) ListModel {
	return ListModel{
		store:  s,
		cursor: 0,
		mode:   ModeNormal,
		width:  w,
		height: h,
	}
}

/* ---------------------------------------------------------------------------
 * Bubble Tea 接口
 * ---------------------------------------------------------------------------*/

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case WOLResultMsg:
		if msg.Err != nil {
			m.showResult("✗ 唤醒失败", fmt.Sprintf("%v", msg.Err), true)
		} else {
			m.showResult("✓ 唤醒成功", fmt.Sprintf("唤醒指令已发送 (%s)", msg.DeviceMAC), false)
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m ListModel) View() string {
	var b strings.Builder

	b.WriteString(RenderBanner(m.width))
	b.WriteString("\n")

	if m.hint != "" {
		b.WriteString(InfoStyle.Render("  "+m.hint) + "\n")
	}

	devices := m.store.List()

	if len(devices) == 0 {
		b.WriteString(PadTop(2))
		b.WriteString(Center(m.width, 3,
			DimRowStyle.Render("(´･_･`)  暂无已保存的唤醒设备")))
		b.WriteString("\n")
		b.WriteString(Center(m.width, 1, HintStyle.Render("按 a 新增第一台设备")))
	} else {
		b.WriteString(m.renderHeader() + "\n")
		b.WriteString(SepStyle.Render(strings.Repeat("─", m.width)) + "\n")

		visible := m.visibleRows()
		offset := m.calcScrollOffset(visible)

		for i := offset; i < min(offset+visible, len(devices)); i++ {
			b.WriteString(m.renderRow(i, devices[i]) + "\n")
		}

		if len(devices) > visible {
			scrollHint := fmt.Sprintf("  ... %d/%d  (j/k 滚动) ...", m.cursor+1, len(devices))
			b.WriteString(lipgloss.NewStyle().Foreground(ColorWarn).Render(scrollHint))
			b.WriteString("\n")
		}
	}

	b.WriteString(RenderStatus(m.status, m.isError))
	b.WriteString("\n")

	// ---- 内联弹窗（删除/唤醒/结果） ----
	if m.mode == ModeConfirm || m.mode == ModeWakeConfirm || m.mode == ModeResult {
		b.WriteString("\n")
		b.WriteString(m.renderPopup())
	}

	// ---- 帮助栏固定在底部 ----
	helpBar := RenderHelpBar(m.width, m.bindings())
	return PinBottom(m.height, b.String(), helpBar)
}

/* ---------------------------------------------------------------------------
 * 按键处理
 * ---------------------------------------------------------------------------*/

func (m ListModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.mode {

	// ---- 删除确认 - Enter/y 确认，n/ESC 取消 ----
	case ModeConfirm:
		switch key {
		case "y", "Y", "enter":
			return m.executeDelete()
		case "n", "N", "esc":
			m.mode = ModeNormal
			m.status = ""
			m.isError = false
		}

	// ---- 唤醒确认 - Enter/y 确认，n/ESC 取消 ----
	case ModeWakeConfirm:
		switch key {
		case "y", "Y", "enter":
			return m.executeWake()
		case "n", "N", "esc":
			m.mode = ModeNormal
			m.status = ""
			m.isError = false
		}

	// ---- 结果弹窗：任意键关闭 ----
	case ModeResult:
		m.mode = ModeNormal
		m.status = ""
		m.isError = false

	// ---- 普通模式：Vim 导航 ----
	case ModeNormal:
		if key != "d" && m.pendingD {
			m.pendingD = false
		}

		switch key {
		case "d":
			if m.pendingD {
				m.pendingD = false
				return m.triggerDeleteConfirm()
			}
			m.pendingD = true

		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)

		case "a":
			return NewFormModel(m.store, m.width, m.height, -1), nil

		case "e":
			if m.store.Count() == 0 {
				m.setStatus("没有可编辑的设备", true)
			} else {
				return NewFormModel(m.store, m.width, m.height, m.cursor), nil
			}

		case "enter", " ":
			if m.store.Count() == 0 {
				m.setStatus("没有可唤醒的设备", true)
			} else {
				m.targetIdx = m.cursor
				m.mode = ModeWakeConfirm
				m.status = ""
				m.isError = false
			}

		case "q", "ctrl+c":
			return m, tea.Quit

		default:
			m.status = ""
			m.isError = false
		}
	}

	return m, nil
}

/* ---------------------------------------------------------------------------
 * 帮助栏
 * ---------------------------------------------------------------------------*/

func (m ListModel) bindings() []KeyBinding {
	switch m.mode {
	case ModeConfirm:
		return []KeyBinding{
			{"Enter/y", "确认删除"}, {"n/ESC", "取消"},
		}
	case ModeWakeConfirm:
		return []KeyBinding{
			{"Enter/y", "确认唤醒"}, {"n/ESC", "取消"},
		}
	case ModeResult:
		return []KeyBinding{
			{"任意键", "关闭"},
		}
	default:
		return []KeyBinding{
			{"j/k", "移动"}, {"Enter", "唤醒"}, {"a", "新增"},
			{"e", "编辑"}, {"dd", "删除"}, {"q", "退出"},
		}
	}
}

/* ---------------------------------------------------------------------------
 * 表格渲染 — 列均分 + 居中对齐
 * ---------------------------------------------------------------------------*/

// colW 计算均分列宽，5 列均分终端宽度（减去箭头 2 字符 + 列间 4 空格）
func (m ListModel) colW() int {
	w := (m.width - 6) / 5
	if w < 6 {
		w = 6
	}
	return w
}

// padCenter 将文本在指定宽度内居中（兼容中文等宽字符）
func padCenter(s string, width int) string {
	dispW := lipgloss.Width(s)
	if dispW >= width {
		return s
	}
	pad := width - dispW
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func (m ListModel) renderHeader() string {
	w := m.colW()
	cells := []string{
		padCenter("序号", w),
		padCenter("名称", w),
		padCenter("MAC 地址", w),
		padCenter("端口", w),
		padCenter("广播地址", w),
	}
	return InfoStyle.Render("  " + strings.Join(cells, " "))
}

func (m ListModel) renderRow(idx int, d store.Device) string {
	w := m.colW()
	cells := []string{
		padCenter(fmt.Sprintf("%d", idx+1), w),
		padCenter(d.Name, w),
		padCenter(d.MAC, w),
		padCenter(fmt.Sprintf("%d", d.Port), w),
		padCenter(d.Address, w),
	}
	arrow := Arrow(idx == m.cursor)
	row := arrow + strings.Join(cells, " ")

	if idx == m.cursor {
		return SelectedStyle.Render(row)
	}
	return NormalRowStyle.Render(row)
}

/* ---------------------------------------------------------------------------
 * 内联弹窗
 * ---------------------------------------------------------------------------*/

func (m ListModel) renderPopup() string {
	// 结果弹窗不需要目标设备
	if m.mode == ModeResult {
		return m.renderResultPopup()
	}

	dev, err := m.store.FindByIndex(m.targetIdx)
	if err != nil {
		return ""
	}

	var title string
	var titleStyle lipgloss.Style
	var lines []string
	var bindings []KeyBinding

	switch m.mode {
	case ModeConfirm:
		title = "🗑  确认删除"
		titleStyle = PopupTitleError
		lines = []string{
			fmt.Sprintf("设备：%s", dev.Name),
			fmt.Sprintf("MAC ：%s", dev.MAC),
		}
		bindings = []KeyBinding{
			{"Enter/y", "确认删除"}, {"n/ESC", "取消"},
		}
	case ModeWakeConfirm:
		title = "⚡ 确认唤醒"
		titleStyle = PopupTitleWarn
		lines = []string{
			fmt.Sprintf("设备：%s", dev.Name),
			fmt.Sprintf("MAC ：%s", dev.MAC),
			fmt.Sprintf("地址：%s:%d", dev.Address, dev.Port),
		}
		bindings = []KeyBinding{
			{"Enter/y", "发送唤醒包"}, {"n/ESC", "取消"},
		}
	}

	popup := RenderPopup(m.width, title, titleStyle, lines, bindings)
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(popup)
}

// renderResultPopup 渲染操作结果弹窗（成功绿/失败红）
func (m ListModel) renderResultPopup() string {
	titleStyle := PopupTitleSuccess
	if m.resultIsErr {
		titleStyle = PopupTitleError
	}
	lines := []string{m.resultMsg}
	bindings := []KeyBinding{{"任意键", "关闭"}}
	popup := RenderPopup(m.width, m.resultTitle, titleStyle, lines, bindings)
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(popup)
}

// showResult 显示操作结果弹窗
func (m *ListModel) showResult(title, msg string, isErr bool) {
	m.resultTitle = title
	m.resultMsg = msg
	m.resultIsErr = isErr
	m.mode = ModeResult
	m.status = ""  // 清除内联状态，避免与弹窗重复
	m.isError = false
}

/* ---------------------------------------------------------------------------
 * 滚动逻辑
 * ---------------------------------------------------------------------------*/

func (m ListModel) visibleRows() int {
	rows := m.height - 8
	if rows < 5 {
		return 5
	}
	return rows
}

func (m ListModel) calcScrollOffset(visible int) int {
	total := m.store.Count()
	if total <= visible {
		return 0
	}
	half := visible / 2
	if m.cursor < half {
		return 0
	}
	if m.cursor >= total-half {
		return total - visible
	}
	return m.cursor - half
}

func (m *ListModel) moveCursor(delta int) {
	total := m.store.Count()
	if total == 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	m.status = ""
	m.isError = false
}

/* ---------------------------------------------------------------------------
 * 操作执行
 * ---------------------------------------------------------------------------*/

func (m *ListModel) setStatus(msg string, isError bool) {
	m.status = msg
	m.isError = isError
}

func (m ListModel) triggerDeleteConfirm() (tea.Model, tea.Cmd) {
	if m.store.Count() == 0 {
		m.setStatus("没有可删除的设备", true)
		return m, nil
	}
	m.targetIdx = m.cursor
	m.mode = ModeConfirm
	m.setStatus("", false)
	return m, nil
}

func (m ListModel) executeDelete() (tea.Model, tea.Cmd) {
	dev, err := m.store.FindByIndex(m.targetIdx)
	if err != nil {
		m.setStatus(err.Error(), true)
		m.mode = ModeNormal
		return m, nil
	}

	if err := m.store.Delete(m.targetIdx); err != nil {
		m.setStatus(err.Error(), true)
		m.mode = ModeNormal
		return m, nil
	}

	total := m.store.Count()
	if m.cursor >= total && total > 0 {
		m.cursor = total - 1
	}

	m.showResult("🗑 删除成功", fmt.Sprintf("设备「%s」已删除", dev.Name), false)

	return m, func() tea.Msg {
		return SaveResultMsg{Err: m.store.Save()}
	}
}

func (m ListModel) executeWake() (tea.Model, tea.Cmd) {
	dev, err := m.store.FindByIndex(m.targetIdx)
	if err != nil {
		m.setStatus(err.Error(), true)
		m.mode = ModeNormal
		return m, nil
	}

	m.mode = ModeNormal
	m.setStatus("正在发送唤醒指令...", false)

	return m, func() tea.Msg {
		err := wol.SendMagicPacket(dev.MAC, dev.Address, dev.Port)
		return WOLResultMsg{
			Err:       err,
			DeviceMAC: dev.MAC,
		}
	}
}
