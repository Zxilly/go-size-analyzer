package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m mainModel) handleKeyEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Switch):
		m.focus = m.nextFocus()
		m.keyMap = m.getKeyMap()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Exit):
		return m, tea.Quit
	}

	var cmd tea.Cmd

	switch m.focus {
	case focusedMain:
		m.leftTable, cmd = m.leftTable.Update(msg)
	case focusedDetail, focusedChildren:
		m.rightDetail, cmd = m.rightDetail.Update(msg)
	}

	return m, cmd
}

func (m mainModel) updateWindowSize(width, height int) (mainModel, tea.Cmd) {
	m.width = width
	m.height = height

	m.leftTable.SetWidth(width / 2)
	m.leftTable.SetColumns(getTableColumns(width))

	m.help.Width = width

	helpHeight := lipgloss.Height(m.help.View(m.getKeyMap()))

	// update the table height accordingly
	m.leftTable.SetHeight(height - 1 - helpHeight - 5)

	// todo: update detail height

	return m, nil
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.updateWindowSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		return m.handleKeyEvent(msg)
	}
	return m, nil
}
