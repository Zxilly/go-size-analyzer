package tui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
)

func getTableStyle(hasChildren bool) table.Styles {
	s := table.DefaultStyles()

	if hasChildren {
		s.Selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("36"))
	}

	return s
}

func (m mainModel) View() tea.View {
	v := tea.NewView(m.renderContent())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
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
	left := tableViewWithScrollbar(m.leftTable)

	right := detailViewWithScrollbar(m.rightDetail)

	borderStyle := baseStyle
	disabledBorderStyle := borderStyle.BorderForeground(lipgloss.Color("241"))

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

	help := m.help.View(m.getKeyMap())

	full := lipgloss.JoinVertical(lipgloss.Top, title, main, help)
	return full
}
