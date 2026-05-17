package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

func (m mainModel) pushParent() mainModel {
	m.parents = append(m.parents, m.leftTable)
	return m
}

func (m mainModel) popParent() mainModel {
	last := len(m.parents) - 1
	m.leftTable = m.parents[last]
	m.parents[last] = table.Model{}
	m.parents = m.parents[:last]
	m.layout = tuiLayout{}
	return m
}

func handleKeyEvent(m mainModel, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Switch):
		m.focus = m.nextFocus()
		m = m.reconcile()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Backward):
		if m.current != nil && m.focus == focusedMain {
			m.current = m.current.parent
			m = m.popParent()
			m = m.reconcile()
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Enter):
		if m.focus != focusedMain {
			break
		}
		if !m.currentSelection().hasChildren() {
			return m, nil
		}
		m = m.pushParent()
		m.current = m.currentSelection()
		m.leftTable = newLeftTable(m.width, m.current.children().ToRows())
		m.layout = tuiLayout{}
		m = m.reconcile()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Help):
		m.help.ShowAll = !m.help.ShowAll
		m = m.reconcile()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Exit):
		return m, tea.Quit
	}

	var cmd tea.Cmd

	switch m.focus {
	case focusedMain:
		if tableHandleKey(&m.leftTable, msg) {
			m = m.reconcile()
			return m, nil
		}
		m.leftTable, cmd = m.leftTable.Update(msg)
		m = m.reconcile()
	case focusedDetail:
		m.rightDetail, cmd = m.rightDetail.Update(msg)
		m = m.reconcile()
	}

	return m, cmd
}

// wheelScrollLines matches bubbles viewport's default MouseWheelDelta.
const wheelScrollLines = 3

func handleMouseWheelEvent(m mainModel, msg tea.MouseWheelMsg) (mainModel, tea.Cmd) {
	// Auto-focus the content area under the mouse so the wheel scrolls that
	// pane without needing a Tab.
	switch {
	case m.layout.leftContent.contains(msg.X, msg.Y):
		m.focus = focusedMain
	case m.layout.rightContent.contains(msg.X, msg.Y):
		m.focus = focusedDetail
	default:
		return m, nil
	}

	switch m.focus {
	case focusedMain:
		switch msg.Button {
		case tea.MouseWheelUp:
			tableScrollBy(&m.leftTable, -wheelScrollLines)
		case tea.MouseWheelDown:
			tableScrollBy(&m.leftTable, wheelScrollLines)
		}
		m = m.reconcile()
		return m, nil
	case focusedDetail:
		var cmd tea.Cmd
		m.rightDetail, cmd = m.rightDetail.Update(msg)
		m = m.reconcile()
		return m, cmd
	}
	return m, nil
}

func (m mainModel) startScrollbarDrag(target scrollbarDragTarget, bar rect, metrics scrollbarMetrics, y int) mainModel {
	grabOffset, ok := scrollbarGrabOffset(metrics, y-bar.y)
	if !ok {
		return m.reconcile()
	}
	m.drag = scrollbarDrag{target: target, grabOffset: grabOffset}
	return m.dragScrollbarTo(y)
}

func (m mainModel) dragScrollbarTo(y int) mainModel {
	var metrics scrollbarMetrics
	var bar rect
	switch m.drag.target {
	case scrollbarDragLeft:
		metrics = tableScrollbarMetrics(m.leftTable)
		bar = m.layout.leftScrollbar
	case scrollbarDragRight:
		metrics = detailScrollbarMetrics(m.rightDetail)
		bar = m.layout.rightScrollbar
	default:
		return m
	}

	offset, ok := scrollbarOffsetForY(metrics, y-bar.y, m.drag.grabOffset)
	if !ok {
		return m
	}

	switch m.drag.target {
	case scrollbarDragLeft:
		tableScrollTo(&m.leftTable, offset)
		m.focus = focusedMain
	case scrollbarDragRight:
		m.rightDetail.viewPort.SetYOffset(offset)
		m.focus = focusedDetail
	}
	m = m.reconcile()
	return m
}

func handleMouseClickEvent(m mainModel, msg tea.MouseClickMsg) (mainModel, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	switch {
	case m.layout.leftScrollbar.contains(msg.X, msg.Y):
		m.focus = focusedMain
		m = m.startScrollbarDrag(scrollbarDragLeft, m.layout.leftScrollbar, tableScrollbarMetrics(m.leftTable), msg.Y)
		return m, nil
	case m.layout.rightScrollbar.contains(msg.X, msg.Y):
		m.focus = focusedDetail
		m = m.startScrollbarDrag(scrollbarDragRight, m.layout.rightScrollbar, detailScrollbarMetrics(m.rightDetail), msg.Y)
		return m, nil
	case m.layout.leftContent.contains(msg.X, msg.Y):
		m.focus = focusedMain
	case m.layout.rightContent.contains(msg.X, msg.Y):
		m.focus = focusedDetail
		m = m.reconcile()
		return m, nil
	default:
		return m, nil
	}

	if !m.layout.leftData.contains(msg.X, msg.Y) {
		m = m.reconcile()
		return m, nil
	}

	rowInVisible := msg.Y - m.layout.leftData.y
	absRow := firstVisibleRow(m.leftTable) + rowInVisible
	rows := m.currentList()
	if absRow < 0 || absRow >= len(rows) {
		m = m.reconcile()
		return m, nil
	}

	tableSelectAt(&m.leftTable, absRow)
	m = m.reconcile()
	return m, nil
}

func handleMouseMotionEvent(m mainModel, msg tea.MouseMotionMsg) (mainModel, tea.Cmd) {
	if m.drag.target == scrollbarDragNone {
		return m, nil
	}
	m = m.dragScrollbarTo(msg.Y)
	return m, nil
}

func handleMouseReleaseEvent(m mainModel, _ tea.MouseReleaseMsg) (mainModel, tea.Cmd) {
	if m.drag.target == scrollbarDragNone {
		return m, nil
	}
	m.drag = scrollbarDrag{}
	m = m.reconcile()
	return m, nil
}

func handleWindowSizeEvent(m mainModel, width, height int) (mainModel, tea.Cmd) {
	m.width = width
	m.height = height
	m.drag = scrollbarDrag{}
	m = m.reconcile()

	return m, tea.ClearScreen
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return handleWindowSizeEvent(m, msg.Width, msg.Height)
	case tea.BackgroundColorMsg:
		m.rightDetail.SetDark(msg.IsDark())
		m = m.reconcile()
		return m, nil
	case tea.KeyPressMsg:
		return handleKeyEvent(m, msg)
	case tea.MouseClickMsg:
		return handleMouseClickEvent(m, msg)
	case tea.MouseMotionMsg:
		return handleMouseMotionEvent(m, msg)
	case tea.MouseReleaseMsg:
		return handleMouseReleaseEvent(m, msg)
	case tea.MouseWheelMsg:
		return handleMouseWheelEvent(m, msg)
	}
	return m, nil
}
