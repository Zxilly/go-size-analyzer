package tui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

const doubleClickThreshold = 500 * time.Millisecond

// nowFunc is overridable in tests.
var nowFunc = time.Now

func (m mainModel) pushParent() mainModel {
	m.parents = append(m.parents, m.leftTable)
	return m
}

func (m mainModel) popParent() mainModel {
	last := len(m.parents) - 1
	m.leftTable = m.parents[last]
	m.parents[last] = hoverTable{}
	m.parents = m.parents[:last]
	m.layout = tuiLayout{}
	return m
}

func (m mainModel) clearHover() mainModel {
	m.leftTable.hoverRow = -1
	return m
}

func (m mainModel) enterSelection() mainModel {
	if m.focus != focusedMain {
		return m
	}
	if !m.currentSelection().hasChildren() {
		return m
	}
	m = m.pushParent()
	m.current = m.currentSelection()
	m.leftTable = newLeftTable(m.width, m.current.children().ToRows())
	m.layout = tuiLayout{}
	m = m.clearHover()
	return m.reconcile()
}

func (m mainModel) goBack() mainModel {
	if m.current == nil {
		return m
	}
	m.current = m.current.parent
	m = m.popParent()
	m = m.clearHover()
	return m.reconcile()
}

func handleKeyEvent(m mainModel, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Switch):
		m.focus = m.nextFocus()
		m = m.reconcile()
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Backward):
		if m.focus == focusedMain {
			m = m.goBack()
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.Enter):
		if m.focus != focusedMain {
			break
		}
		return m.enterSelection(), nil
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
		m.leftTable.Model, cmd = m.leftTable.Update(msg)
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
		// The pointer hasn't moved but the row under it has, since the
		// visible window shifted. Recompute hover from msg.X/msg.Y so the
		// highlight follows the data the user is now pointing at instead
		// of staying on the old row index.
		m.leftTable.hoverRow = m.hoverRowAt(msg.X, msg.Y)
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
		metrics = tableScrollbarMetrics(m.leftTable.Model)
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
	if msg.Button == tea.MouseRight {
		if m.layout.leftContent.contains(msg.X, msg.Y) {
			m.focus = focusedMain
			return m.goBack(), nil
		}
		return m, nil
	}
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	switch {
	case m.layout.leftScrollbar.contains(msg.X, msg.Y):
		m.focus = focusedMain
		m = m.startScrollbarDrag(scrollbarDragLeft, m.layout.leftScrollbar, tableScrollbarMetrics(m.leftTable.Model), msg.Y)
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
	absRow := firstVisibleRow(m.leftTable.Model) + rowInVisible
	rows := m.currentList()
	if absRow < 0 || absRow >= len(rows) {
		m = m.reconcile()
		return m, nil
	}

	tableSelectAt(&m.leftTable, absRow)

	now := nowFunc()
	if !m.lastClickAt.IsZero() &&
		m.lastClickRow == absRow &&
		now.Sub(m.lastClickAt) <= doubleClickThreshold {
		m.lastClickAt = time.Time{}
		return m.enterSelection(), nil
	}
	m.lastClickAt = now
	m.lastClickRow = absRow

	m = m.reconcile()
	return m, nil
}

func handleMouseMotionEvent(m mainModel, msg tea.MouseMotionMsg) (mainModel, tea.Cmd) {
	if m.drag.target != scrollbarDragNone {
		m = m.dragScrollbarTo(msg.Y)
		return m, nil
	}
	return m.updateHover(msg.X, msg.Y), nil
}

// hoverRowAt returns -1 outside leftData or past the last row.
func (m mainModel) hoverRowAt(x, y int) int {
	if !m.layout.leftData.contains(x, y) {
		return -1
	}
	absRow := firstVisibleRow(m.leftTable.Model) + (y - m.layout.leftData.y)
	if absRow < 0 || absRow >= len(m.currentList()) {
		return -1
	}
	return absRow
}

// updateHover skips reconcile when the row didn't change so we don't redraw
// the whole table viewport on every motion event.
func (m mainModel) updateHover(x, y int) mainModel {
	row := m.hoverRowAt(x, y)
	if row == m.leftTable.hoverRow {
		return m
	}
	m.leftTable.hoverRow = row
	return m.reconcile()
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
	m = m.clearHover()
	m = m.reconcile()

	return m, tea.ClearScreen
}

// applyHelpMode bumps helpMode based on the kind of input event so the
// compact bar matches the modality the user just used. Hover/release/etc.
// don't flip it — only intentional key or mouse actions.
func (m mainModel) applyHelpMode(msg tea.Msg) mainModel {
	switch msg.(type) {
	case tea.KeyPressMsg:
		m.helpMode = helpModeKeyboard
	case tea.MouseClickMsg, tea.MouseWheelMsg:
		m.helpMode = helpModeMouse
	}
	return m
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m = m.applyHelpMode(msg)

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
