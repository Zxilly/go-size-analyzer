package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// todo: store the parent cursor to recover the cursor position when going back
func updateCurrent(m *mainModel, wrapper *wrapper) {
	m.current = wrapper
	m.leftTable.SetCursor(0)
	m.leftTable.SetRows(m.currentList().ToRows())
	m.rightDetail.viewPort.SetContent(m.currentSelection().Description())
}

func (m mainModel) handleKeyEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Switch):
		m.focus = m.nextFocus()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Backward):
		if m.current != nil && m.focus == focusedMain {
			updateCurrent(&m, m.current.parent)
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Enter):
		if m.currentSelection().hasChildren() && m.focus == focusedMain {
			updateCurrent(&m, m.currentSelection())
		}
	case key.Matches(msg, DefaultKeyMap.Exit):
		return m, tea.Quit
	}

	var cmd tea.Cmd

	switch m.focus {
	case focusedMain:
		m.leftTable, cmd = m.leftTable.Update(msg)
		m.rightDetail.viewPort.SetContent(m.currentSelection().Description())
	case focusedDetail:
		m.rightDetail, cmd = m.rightDetail.Update(msg)
	}

	return m, cmd
}

func (m mainModel) handleWindowSizeEvent(width, height int) (mainModel, tea.Cmd) {
	m.width = width
	m.height = height

	m.leftTable.SetWidth(width / 2)
	m.leftTable.SetColumns(getTableColumns(width))

	m.help.Width = width

	helpHeight := lipgloss.Height(m.help.View(m.getKeyMap()))

	const headerHeight = 1
	const nameHeight = 1

	// update the table height accordingly
	m.leftTable.SetHeight(height - helpHeight - headerHeight - nameHeight - 1)

	m.rightDetail.viewPort.Height = height - helpHeight - nameHeight - 2
	m.rightDetail.viewPort.Width = width - width/2 - 1

	return m, tea.ClearScreen
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeEvent(msg.Width, msg.Height)
	case tea.KeyMsg:
		return m.handleKeyEvent(msg)
	}
	return m, nil
}
