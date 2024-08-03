package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m mainModel) pushParent(cursor table.Model) mainModel {
	m.parents = append(m.parents, cursor)
	return m
}

func (m mainModel) popParent() (mainModel, table.Model) {
	old := m.parents[len(m.parents)-1]
	m.parents = m.parents[:len(m.parents)-1]
	return m, old
}

func (m mainModel) updateDetail() mainModel {
	m.rightDetail.viewPort.SetContent(m.currentSelection().Description())
	return m
}

func handleKeyEvent(m mainModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Switch):
		m.focus = m.nextFocus()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Backward):
		if m.current != nil && m.focus == focusedMain {
			m.current = m.current.parent

			var parent table.Model
			m, parent = m.popParent()
			m.leftTable = parent
			m = m.updateDetail()
			m, _ = handleWindowSizeEvent(m, m.width, m.height)
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Enter):
		if m.currentSelection().hasChildren() && m.focus == focusedMain {
			m = m.pushParent(m.leftTable)
			m.current = m.currentSelection()
			m.leftTable = newLeftTable(m.width, m.current.children().ToRows())
			m = m.updateDetail()
			m, _ = handleWindowSizeEvent(m, m.width, m.height)
		}
	case key.Matches(msg, DefaultKeyMap.Exit):
		return m, tea.Quit
	}

	var cmd tea.Cmd

	switch m.focus {
	case focusedMain:
		m.leftTable, cmd = m.leftTable.Update(msg)
		m = m.updateDetail()
	case focusedDetail:
		m.rightDetail, cmd = m.rightDetail.Update(msg)
	}

	return m, cmd
}

func tableHandleMouseEvent(t table.Model, msg tea.MouseMsg) (table.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		t.MoveUp(1)
	case tea.MouseButtonWheelDown:
		t.MoveDown(1)
	default:
		return t, nil
	}

	return t, nil
}

func handleMouseEvent(m mainModel, msg tea.MouseMsg) (mainModel, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	var cmd tea.Cmd

	switch m.focus {
	case focusedMain:
		m.leftTable, cmd = tableHandleMouseEvent(m.leftTable, msg)
		m = m.updateDetail()
	case focusedDetail:
		m.rightDetail, cmd = m.rightDetail.Update(msg)
	}

	return m, cmd
}

func handleWindowSizeEvent(m mainModel, width, height int) (mainModel, tea.Cmd) {
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
		return handleWindowSizeEvent(m, msg.Width, msg.Height)
	case tea.KeyMsg:
		return handleKeyEvent(m, msg)
	case tea.MouseMsg:
		return handleMouseEvent(m, msg)
	}
	return m, nil
}
