package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

func (m mainModel) View() string {
	if m.width < 70 || m.height < 20 {
		return wordwrap.String(
			fmt.Sprintf("Your terminal window is too small. "+
				"Please make it at least 70x20 and try again. Current size: %d x %d", m.width, m.height),
			m.width)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Width(m.width).
		MaxWidth(m.width).
		Align(lipgloss.Center).
		Render(m.title())

	// Render the left table
	left := m.leftTable.View()

	// Render the right detail
	right := m.rightDetail.View()

	borderStyle := baseStyle.Width(m.width / 2)
	disabledBorderStyle := borderStyle.BorderForeground(lipgloss.Color("241"))

	switch m.focus {
	case focusedMain:
		left = borderStyle.Render(left)
		right = disabledBorderStyle.Render(right)
	case focusedDetail:
		left = disabledBorderStyle.Render(left)
		right = borderStyle.Render(right)
	}

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	help := m.help.View(m.getKeyMap())

	full := lipgloss.JoinVertical(lipgloss.Top, title, main, help)
	return full
}
