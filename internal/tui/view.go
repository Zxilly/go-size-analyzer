package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
)

func getTableStyle(hasChildren bool) table.Styles {
	s := table.DefaultStyles()

	if hasChildren {
		s.Selected = lipgloss.NewStyle().Bold(true).Foreground(colorSelected)
	}

	return s
}

var hoverRowStyle = lipgloss.NewStyle().Background(colorHoverBg)

func (m mainModel) View() tea.View {
	v := tea.NewView(m.renderContent())
	v.AltScreen = true
	// AllMotion is required to receive hover (motion without a button held).
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func (m mainModel) renderContent() string {
	if m.width < minTerminalWidth || m.height < minTerminalHeight {
		return wordwrap.String(
			fmt.Sprintf("Your terminal window is too small. "+
				"Please make it at least %dx%d and try again. Current size: %d x %d",
				minTerminalWidth, minTerminalHeight, m.width, m.height),
			m.width)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Width(m.width).
		MaxWidth(m.width).
		Align(lipgloss.Center).
		Render(m.title())

	// SetStyles intentionally lives in reconcileSelection, not here: bubbles'
	// SetStyles calls UpdateViewport which re-anchors start/end around the
	// cursor and would wipe any wheel-scroll state on the very next render.
	left := tableViewWithScrollbar(m.leftTable.Model)

	right := detailViewWithScrollbar(m.rightDetail)

	borderStyle := baseStyle
	disabledBorderStyle := borderStyle.BorderForeground(colorBorderDisabled)

	switch m.focus {
	case focusedMain:
		left = borderStyle.Width(m.layout.leftPane.w).Render(left)
		right = disabledBorderStyle.Width(m.layout.rightPane.w).Render(right)
	case focusedDetail:
		left = disabledBorderStyle.Width(m.layout.leftPane.w).Render(left)
		right = borderStyle.Width(m.layout.rightPane.w).Render(right)
	default:
	}

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	help := m.renderHelpBar()

	full := lipgloss.JoinVertical(lipgloss.Top, title, main, help)

	if m.help.ShowAll {
		dialog := m.renderHelpDialog()
		base := lipgloss.NewLayer(full)
		over := lipgloss.NewLayer(dialog).
			X(m.layout.helpDialog.x).
			Y(m.layout.helpDialog.y).
			Z(1)
		return lipgloss.NewCompositor(base, over).Render()
	}
	return full
}

func (m mainModel) renderHelpBar() string {
	if m.helpMode == helpModeKeyboard {
		return m.help.View(m.getKeyMap())
	}
	// Mouse mode: short-form renderer with mouse bindings + keyboard essentials.
	bindings := append(append([]key.Binding{}, mouseShortList...),
		DefaultKeyMap.Help, DefaultKeyMap.Exit)
	return m.help.ShortHelpView(bindings)
}

var helpDialogBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 1)

func (m mainModel) renderHelpDialog() string {
	inner := m.layout.helpDialog.w - helpDialogBorder.GetHorizontalFrameSize()
	if inner < 1 {
		inner = 1
	}

	keyboardCol := m.renderHelpColumn("Keyboard", m.flatKeyboardBindings())
	mouseCol := m.renderHelpColumn("Mouse", mouseFullList)

	// Odd inner: the second column absorbs the extra cell.
	colWidth := inner / 2
	keyboardCol = lipgloss.NewStyle().Width(colWidth).Render(keyboardCol)
	mouseCol = lipgloss.NewStyle().Width(inner - colWidth).Render(mouseCol)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, keyboardCol, mouseCol)

	dimStyle := lipgloss.NewStyle().Foreground(colorBorderDisabled)
	closeBtn := lipgloss.PlaceHorizontal(inner, lipgloss.Right, dimStyle.Render(helpDialogCloseLabel))
	title := lipgloss.NewStyle().Bold(true).Render("Help")
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(inner-helpDialogCloseW).Render(title),
		lipgloss.NewStyle().Width(helpDialogCloseW).Render(closeBtn))
	footer := dimStyle.Width(inner).Align(lipgloss.Center).Render(helpDialogCloseHint())

	body := lipgloss.JoinVertical(lipgloss.Top, header, "", columns, "", footer)
	return helpDialogBorder.Width(inner).Render(body)
}

func (m mainModel) flatKeyboardBindings() []key.Binding {
	_, long := m.keyboardBindings()
	return slices.Concat(long...)
}

func (m mainModel) renderHelpColumn(title string, bindings []key.Binding) string {
	enabled := bindings[:0:0]
	maxKey := 0
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		enabled = append(enabled, b)
		if w := lipgloss.Width(b.Help().Key); w > maxKey {
			maxKey = w
		}
	}

	lines := make([]string, 0, len(enabled)+1)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render(title))
	for _, b := range enabled {
		h := b.Help()
		keyCell := lipgloss.NewStyle().Width(maxKey).Render(m.help.Styles.FullKey.Render(h.Key))
		lines = append(lines, keyCell+" "+m.help.Styles.FullDesc.Render(h.Desc))
	}
	return strings.Join(lines, "\n")
}

// helpDialogCloseHint derives the close-dialog instruction from the live key
// bindings so re-binding `?` or `Esc` updates the message automatically.
func helpDialogCloseHint() string {
	return fmt.Sprintf("%s / %s / right-click / click outside to close",
		DefaultKeyMap.Help.Help().Key,
		DefaultKeyMap.Backward.Help().Key)
}
