/*
 * ui/form.go — 新增/编辑设备表单页
 *
 * 职责：
 *   1. 4 个字段一屏展示（名称/MAC/端口/广播地址），Tab/j/k 切换焦点
 *   2. 实时校验当前字段，Enter 提交时全字段校验
 *   3. 编辑模式下预填已有数据
 */

package ui

import (
	"fmt"
	"strconv"
	"strings"

	"WakeUp/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

/* ---------------------------------------------------------------------------
 * FormModel
 * ---------------------------------------------------------------------------*/

type FormModel struct {
	store        *store.Store
	inputs       [4]textinput.Model // 0=名称 1=MAC 2=端口 3=地址
	focusIdx     int                // 当前聚焦字段
	errors       [4]string          // 各字段校验错误
	editingIndex int                // -1=新增, >=0=编辑
	width        int
	height       int
}

// field labels
var fieldLabels = [4]string{"设备名称", "MAC 地址", "端口", "广播地址"}

// NewFormModel 创建表单模型
func NewFormModel(s *store.Store, w, h int, editingIndex int) FormModel {
	m := FormModel{
		store:        s,
		focusIdx:     0,
		editingIndex: editingIndex,
		width:        w,
		height:       h,
	}

	// 初始化 4 个输入框
	placeholders := [4]string{
		"输入唯一设备别名，如：NAS、台式机",
		"XX:XX:XX:XX:XX:XX 或 XX-XX-XX-XX-XX-XX",
		"默认 9",
		"默认 255.255.255.255",
	}
	defaults := [4]string{"", "", "9", "255.255.255.255"}

	for i := 0; i < 4; i++ {
		m.inputs[i] = textinput.New()
		m.inputs[i].Placeholder = placeholders[i]
		m.inputs[i].CharLimit = 64
		m.inputs[i].Width = min(w-10, 50)

		// 预填默认值或已有数据
		if editingIndex >= 0 {
			if dev, err := s.FindByIndex(editingIndex); err == nil {
				switch i {
				case 0:
					m.inputs[i].SetValue(dev.Name)
				case 1:
					m.inputs[i].SetValue(dev.MAC)
				case 2:
					m.inputs[i].SetValue(strconv.Itoa(dev.Port))
				case 3:
					m.inputs[i].SetValue(dev.Address)
				}
			}
		} else if defaults[i] != "" {
			m.inputs[i].SetValue(defaults[i])
		}
	}

	m.inputs[0].Focus()
	return m
}

/* ---------------------------------------------------------------------------
 * Bubble Tea 接口
 * ---------------------------------------------------------------------------*/

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for i := 0; i < 4; i++ {
			m.inputs[i].Width = min(m.width-10, 50)
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// 非按键消息（Blink 等）传给聚焦输入框
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	return m, cmd
}

func (m FormModel) View() string {
	var b strings.Builder

	// 标题
	title := "➕ 新增唤醒设备"
	if m.editingIndex >= 0 {
		title = "✏️  编辑设备信息"
	}
	b.WriteString(RenderTitle(title, m.width))
	b.WriteString("\n\n")

	// 4 个字段
	for i := 0; i < 4; i++ {
		// 标签（聚焦字段高亮）
		label := fmt.Sprintf("  %s", fieldLabels[i])
		if i == m.focusIdx {
			b.WriteString(LabelStyle.Render(label))
		} else {
			b.WriteString(HintStyle.Render(label))
		}
		b.WriteString("\n")

		// 输入框
		b.WriteString("  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")

		// 错误提示
		if m.errors[i] != "" {
			b.WriteString(ErrorStyle.Render("  ✗ " + m.errors[i]))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// 提示
	b.WriteString(HintStyle.Render("  💡 端口默认9，广播地址默认255.255.255.255，留空即用默认值"))
	b.WriteString("\n")

	// 状态
	// 帮助栏
	bindings := []KeyBinding{
		{"Tab/j↓/k↑", "切换字段"},
		{"Enter", "保存"},
		{"ESC", "返回"},
	}
	helpBar := RenderHelpBar(m.width, bindings)
	return PinBottom(m.height, b.String(), helpBar)
}

/* ---------------------------------------------------------------------------
 * 按键处理
 * ---------------------------------------------------------------------------*/

func (m FormModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {

	// ---- 导航 ----
	case "tab", "enter":
		return m.nextField()

	case "shift+tab":
		return m.prevField()

	case "j", "down":
		return m.focusNext()

	case "k", "up":
		return m.focusPrev()

	// ---- 退出 ----
	case "esc":
		return m.goBack(), nil

	case "ctrl+c":
		return m, tea.Quit

	// ---- 输入：传给聚焦框 + 实时校验 ----
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		m.validateField(m.focusIdx)
		return m, cmd
	}
}

/* ---------------------------------------------------------------------------
 * 焦点移动
 * ---------------------------------------------------------------------------*/

func (m FormModel) nextField() (tea.Model, tea.Cmd) {
	// 先校验当前字段
	if err := m.validateFieldStrict(m.focusIdx); err != nil {
		m.errors[m.focusIdx] = err.Error()
		return m, nil
	}

	if m.focusIdx == 3 {
		// 最后一个字段，提交保存
		return m.save()
	}

	m.inputs[m.focusIdx].Blur()
	m.focusIdx++
	m.inputs[m.focusIdx].Focus()
	return m, nil
}

func (m FormModel) prevField() (tea.Model, tea.Cmd) {
	if m.focusIdx > 0 {
		m.inputs[m.focusIdx].Blur()
		m.focusIdx--
		m.inputs[m.focusIdx].Focus()
	}
	return m, nil
}

func (m FormModel) focusNext() (tea.Model, tea.Cmd) {
	if m.focusIdx < 3 {
		m.inputs[m.focusIdx].Blur()
		m.focusIdx++
		m.inputs[m.focusIdx].Focus()
	}
	return m, nil
}

func (m FormModel) focusPrev() (tea.Model, tea.Cmd) {
	if m.focusIdx > 0 {
		m.inputs[m.focusIdx].Blur()
		m.focusIdx--
		m.inputs[m.focusIdx].Focus()
	}
	return m, nil
}

/* ---------------------------------------------------------------------------
 * 校验
 * ---------------------------------------------------------------------------*/

// validateField 宽松校验（实时，不阻塞输入）
func (m *FormModel) validateField(idx int) {
	val := m.inputs[idx].Value()

	switch idx {
	case 0: // 名称
		if val == "" {
			m.errors[idx] = ""
		} else if err := store.ValidateName(val); err != nil {
			m.errors[idx] = err.Error()
		} else {
			m.errors[idx] = ""
		}

	case 1: // MAC
		if val == "" {
			m.errors[idx] = ""
			return
		}
		_, err := store.ValidateMAC(val)
		if err == store.ErrMACPartial {
			m.errors[idx] = "" // 未完成，不报错
		} else if err != nil {
			m.errors[idx] = err.Error()
		} else {
			m.errors[idx] = ""
		}

	case 2: // 端口
		if val == "" {
			m.errors[idx] = ""
			return
		}
		p, err := strconv.Atoi(val)
		if err != nil {
			m.errors[idx] = store.ErrPortInvalid.Error()
		} else if err := store.ValidatePort(p); err != nil {
			m.errors[idx] = err.Error()
		} else {
			m.errors[idx] = ""
		}

	case 3: // 地址
		if val == "" {
			m.errors[idx] = ""
			return
		}
		if err := store.ValidateIP(val); err != nil {
			m.errors[idx] = err.Error()
		} else {
			m.errors[idx] = ""
		}
	}
}

// validateFieldStrict 严格校验（Enter 切换字段时，不允许空必填项）
func (m *FormModel) validateFieldStrict(idx int) error {
	val := m.inputs[idx].Value()

	switch idx {
	case 0:
		return store.ValidateName(val)
	case 1:
		if val == "" {
			return store.ErrMACInvalid
		}
		_, err := store.ValidateMAC(val)
		return err
	case 2, 3:
		// 端口和地址可空，提交时填默认值
		return nil
	}
	return nil
}

/* ---------------------------------------------------------------------------
 * 保存 / 返回
 * ---------------------------------------------------------------------------*/

func (m FormModel) save() (tea.Model, tea.Cmd) {
	// 全字段严格校验
	name := m.inputs[0].Value()
	if err := store.ValidateName(name); err != nil {
		m.errors[0] = err.Error()
		return m, nil
	}

	mac := m.inputs[1].Value()
	if mac == "" {
		m.errors[1] = store.ErrMACInvalid.Error()
		return m, nil
	}
	normalizedMAC, err := store.ValidateMAC(mac)
	if err != nil {
		m.errors[1] = err.Error()
		return m, nil
	}

	portStr := m.inputs[2].Value()
	if portStr == "" {
		portStr = "9"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || store.ValidatePort(port) != nil {
		m.errors[2] = store.ErrPortInvalid.Error()
		return m, nil
	}

	address := m.inputs[3].Value()
	if address == "" {
		address = "255.255.255.255"
	}
	if err := store.ValidateIP(address); err != nil {
		m.errors[3] = err.Error()
		return m, nil
	}

	dev := store.Device{
		Name:    name,
		MAC:     normalizedMAC,
		Port:    port,
		Address: address,
	}

	var resultTitle, resultMsg string
	var resultIsErr bool

	if m.editingIndex >= 0 {
		if err := m.store.Update(m.editingIndex, dev); err != nil {
			resultTitle = "✗ 更新失败"
			resultMsg = err.Error()
			resultIsErr = true
		} else {
			resultTitle = "✓ 更新成功"
			resultMsg = fmt.Sprintf("设备「%s」已更新", name)
		}
	} else {
		if err := m.store.Add(dev); err != nil {
			resultTitle = "✗ 保存失败"
			resultMsg = err.Error()
			resultIsErr = true
		} else {
			resultTitle = "✓ 保存成功"
			resultMsg = fmt.Sprintf("设备「%s」已保存", name)
		}
	}

	// 返回列表页，带结果弹窗
	list := NewListModel(m.store, m.width, m.height)
	list.showResult(resultTitle, resultMsg, resultIsErr)

	return list, func() tea.Msg {
		return SaveResultMsg{Err: m.store.Save()}
	}
}

func (m FormModel) goBack() tea.Model {
	return NewListModel(m.store, m.width, m.height)
}
