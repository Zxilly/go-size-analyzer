package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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

	// Render the title
	// padding title length to full width
	//title := m.title()
	//
	//// Render the left table
	//left := m.leftTable.View()
	//
	//// Render the right detail
	//right := m.rightDetail.View()
	//
	//main := lipgloss.JoinHorizontal(lipgloss.Center, left, right)
	//
	//help := m.help.View(m.keyMap)
	//
	//full := lipgloss.JoinVertical(lipgloss.Left, title, main, help)
	return m.leftTable.View()
}
