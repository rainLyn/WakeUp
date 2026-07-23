/*
 * ui/ui.go — TUI 界面共享基础设施
 *
 * 职责：
 *   1. 全局 Lipgloss 样式定义（完全使用终端默认颜色，自适应深浅主题）
 *   2. 通用 UI 渲染函数（标题栏、帮助栏、状态提示、居底固定、弹窗）
 *   3. Vim 模态常量与按键绑定定义
 *
 * 配色哲学：
 *   - 不使用硬编码 RGB 色值
 *   - 复用终端 16 色标准调色板 + AdaptiveColor 自动适配
 *   - 保证黑底/白底/自定义主题下均正常显示
 */

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

/* ---------------------------------------------------------------------------
 * Vim 模态常量
 * ---------------------------------------------------------------------------*/

const (
	ModeNormal      = iota // 普通浏览模式：j/k 导航、dd 删除、Enter 唤醒
	ModeInsert             // 编辑输入模式：文本输入、ESC 退出
	ModeConfirm            // 删除确认弹窗：Enter/y 确认
	ModeWakeConfirm        // 唤醒确认弹窗：Enter/y 确认
	ModeResult             // 操作结果弹窗：任意键关闭
)

/* ---------------------------------------------------------------------------
 * Lipgloss 样式 — 全部使用终端原生配色
 * ---------------------------------------------------------------------------*/

var (
	// 主色调 — 随终端主题自动适配明暗
	ColorFg       = lipgloss.AdaptiveColor{Light: "0", Dark: "7"} // 主文字色
	ColorDim      = lipgloss.AdaptiveColor{Light: "8", Dark: "8"} // 次要文字
	ColorSuccess  = lipgloss.AdaptiveColor{Light: "2", Dark: "2"} // 成功（绿）
	ColorError    = lipgloss.AdaptiveColor{Light: "1", Dark: "1"} // 错误（红）
	ColorAccent   = lipgloss.AdaptiveColor{Light: "4", Dark: "4"} // 强调（蓝）
	ColorWarn     = lipgloss.AdaptiveColor{Light: "3", Dark: "3"} // 警告（黄）
	ColorBgSubtle = lipgloss.AdaptiveColor{Light: "7", Dark: "0"} // 微妙背景

	// Banner 样式：居中、无边框
	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1)

	// 选中行样式：加粗 + 强调色
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// 偶数行（主色）
	EvenRowStyle = lipgloss.NewStyle().
			Foreground(ColorFg)

	// 奇数行（略暗，形成斑马纹）
	OddRowStyle = lipgloss.NewStyle().
			Foreground(ColorDim).Faint(true)

	// 通用行样式（菜单等）
	NormalRowStyle = lipgloss.NewStyle().
			Foreground(ColorFg)

	// 表头样式
	DimRowStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// 底部分隔栏样式
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorDim).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(ColorAccent).
			Padding(0, 1)

	// 表格分隔线样式
	SepStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// 成功 / 错误状态提示
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).Bold(true)

	// 提示/信息样式
	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// 加载中样式
	LoadingStyle = lipgloss.NewStyle().
			Foreground(ColorWarn)

	// 表单标签样式
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).Bold(true)

	// 表单默认值提示
	HintStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// 弹窗样式（唤醒确认/删除确认）
	PopupBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(1, 3)

	PopupTitleStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).Bold(true)

	PopupTitleSuccess = lipgloss.NewStyle().
				Foreground(ColorSuccess).Bold(true)

	PopupTitleError = lipgloss.NewStyle().
				Foreground(ColorError).Bold(true)

	PopupTitleWarn = lipgloss.NewStyle().
				Foreground(ColorWarn).Bold(true)

	PopupTextStyle = lipgloss.NewStyle().
				Foreground(ColorFg)
)

/* ---------------------------------------------------------------------------
 * KeyBinding — 按键绑定
 * ---------------------------------------------------------------------------*/

// KeyBinding 表示一个按键及其功能说明
type KeyBinding struct {
	Key  string
	Desc string
}

// RenderHelpBar 渲染底部按键提示栏（键名着色，说明灰色）
func RenderHelpBar(width int, bindings []KeyBinding) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorDim)

	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.Key)+descStyle.Render(" "+b.Desc))
	}
	line := strings.Join(parts, descStyle.Render("  │  "))
	return HelpStyle.Width(width).Render(line)
}

/* ---------------------------------------------------------------------------
 * 通用渲染函数
 * ---------------------------------------------------------------------------*/

// RenderBanner 渲染多彩 "⚡ WakeUp!" 标题
// ⚡ 用黄色(警告色)，WakeUp 用蓝色(强调色)
func RenderBanner(width int) string {
	bolt := lipgloss.NewStyle().Foreground(ColorWarn).Bold(true).Render("⚡")
	name := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("WakeUp!")
	return BannerStyle.Width(width).Render(bolt + name)
}

// RenderTitle 渲染通用页面标题（表单/弹窗等）
func RenderTitle(text string, width int) string {
	return BannerStyle.Width(width).Render(text)
}

// RenderStatus 渲染状态提示（成功/错误）
func RenderStatus(msg string, isError bool) string {
	if msg == "" {
		return ""
	}
	if isError {
		return "\n" + ErrorStyle.Render(" ✗ "+msg)
	}
	return "\n" + SuccessStyle.Render(" ✓ "+msg)
}

// Center 居中布局
func Center(width, height int, content string) string {
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// PadTop 顶部留白
func PadTop(n int) string {
	return strings.Repeat("\n", n)
}

// PinBottom 将 footer 固定在屏幕底部
// content 和 footer 之间用空白填充，确保 footer 始终在 height 行的底部
func PinBottom(height int, content string, footer string) string {
	contentH := lipgloss.Height(content)
	footerH := lipgloss.Height(footer)
	pad := max(0, height-contentH-footerH)
	return content + strings.Repeat("\n", pad) + "\n" + footer
}

// RenderPopup 渲染一个居中的弹窗
// titleStyle 控制标题颜色（成功绿/错误红/警告黄/强调蓝）
func RenderPopup(width int, title string, titleStyle lipgloss.Style, lines []string, bindings []KeyBinding) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	for _, l := range lines {
		b.WriteString(PopupTextStyle.Render(l))
		b.WriteString("\n")
	}

	// 弹窗内按键提示
	if len(bindings) > 0 {
		parts := make([]string, len(bindings))
		for i, kb := range bindings {
			parts[i] = fmt.Sprintf("%s %s", kb.Key, kb.Desc)
		}
		b.WriteString("\n")
		b.WriteString(HintStyle.Render(strings.Join(parts, "  │  ")))
	}

	popupContent := b.String()
	popupWidth := lipgloss.Width(popupContent) + 8
	if popupWidth > width-4 {
		popupWidth = width - 4
	}

	return PopupBorderStyle.Width(popupWidth).Render(popupContent)
}

// Arrow 选中指示箭头
func Arrow(isSelected bool) string {
	if isSelected {
		return "▶ "
	}
	return "  "
}

/* ---------------------------------------------------------------------------
 * 通用消息类型
 * ---------------------------------------------------------------------------*/

// SaveResultMsg 异步保存完成消息
type SaveResultMsg struct {
	Err error
}

// WOLResultMsg 异步 WOL 发送结果
type WOLResultMsg struct {
	Err       error
	DeviceMAC string
}
