package tui

import "charm.land/lipgloss/v2"

func (m mainModel) reconcile() mainModel {
	m = m.reconcileLayout()
	m = m.reconcileSelection()
	return m
}

func (m mainModel) reconcileLayout() mainModel {
	m.help.SetWidth(m.width)
	helpHeight := lipgloss.Height(m.help.View(m.getKeyMap()))
	next := computeLayout(m.width, m.height, helpHeight)
	if next == m.layout {
		return m
	}

	leftTop := firstVisibleRow(m.leftTable.Model)
	if next.leftTable.w != m.layout.leftTable.w {
		m.leftTable.SetWidth(next.leftTable.w)
		m.leftTable.SetColumns(getTableColumnsForTableWidth(next.leftTable.w))
	}
	if next.leftTable.h != m.layout.leftTable.h {
		m.leftTable.SetHeight(next.leftTable.h)
	}
	tableScrollTo(&m.leftTable, leftTop)

	if next.rightDetail.w != m.layout.rightDetail.w {
		m.rightDetail.SetWidth(next.rightDetail.w)
	}
	if next.rightDetail.h != m.layout.rightDetail.h {
		m.rightDetail.SetHeight(next.rightDetail.h)
	}

	m.layout = next
	return m
}

func (m mainModel) reconcileSelection() mainModel {
	sel := m.currentSelection()
	m.rightDetail.SetMarkdown(sel.Description())
	tableSetStyles(&m.leftTable, getTableStyle(sel.hasChildren()))
	return m
}
