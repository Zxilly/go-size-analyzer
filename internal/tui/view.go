package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
	"strings"
)

func (m mainModel) emptyView(content string) string {
	msg := wordwrap.String(content, m.width)
	// padding to full screen
	line := strings.Repeat(" ", m.width) + "\n"
	msgHeight := lipgloss.Height(msg)
	paddingHeight := m.height - msgHeight
	padding := strings.Repeat(line, paddingHeight)
	msg = msg + padding
	return msg
}

func (m mainModel) View() string {
	if m.width < 70 || m.height < 20 {
		return m.emptyView(fmt.Sprintf("Your terminal window is too small. "+
			"Please make it at least 70x20 and try again. Current size: %d x %d", m.width, m.height))
	}

	lipgloss.SetColorProfile(termenv.ANSI)

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
	disabledBorderStyle := borderStyle.Foreground(lipgloss.Color("241"))

	switch m.focus {
	case focusedMain:
		left = borderStyle.Render(left)
		right = disabledBorderStyle.Render(right)
	case focusedDetail:
		left = disabledBorderStyle.Render(left)
		right = borderStyle.Render(right)
	}

	main := lipgloss.JoinHorizontal(lipgloss.Center, left, right)

	help := m.help.View(m.keyMap)

	full := lipgloss.JoinVertical(lipgloss.Top, title, main, help)
	return full
}
