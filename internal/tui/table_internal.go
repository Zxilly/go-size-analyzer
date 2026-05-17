package tui

import (
	"unsafe"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// tableInternal mirrors charm.land/bubbles/v2.table.Model's exact field layout
// so we can read the private viewport's YOffset. Upstream doesn't expose it,
// but we need it to map a mouse click's screen Y to an absolute row index for
// click-to-select. Pinned to bubbles v2.1.0 via go.mod; a layout drift on
// upgrade would corrupt this read — table_internal_test guards against that.
type tableInternal struct {
	KeyMap table.KeyMap
	Help   help.Model

	cols   []table.Column
	rows   []table.Row
	cursor int
	focus  bool
	styles table.Styles

	viewport viewport.Model
	start    int
	end      int
}

// hoverTable carries the embedded table.Model together with the row index
// currently under the mouse (-1 = none). Helpers that rebuild the table
// window take *hoverTable so they can read hover natively, removing the
// need for a package-level state mirror.
type hoverTable struct {
	table.Model
	hoverRow int
}

func tableInternals(t *table.Model) *tableInternal {
	if unsafe.Sizeof(*t) != unsafe.Sizeof(tableInternal{}) {
		panic("bubbles table.Model layout changed")
	}
	return (*tableInternal)(unsafe.Pointer(t))
}

// firstVisibleRow returns the absolute index of the row currently rendered at
// the top of the table's data area.
func firstVisibleRow(t table.Model) int {
	p := tableInternals(&t)
	return p.start + p.viewport.YOffset()
}

func clampInt(n, lo, hi int) int {
	return min(max(n, lo), hi)
}

func renderTableRow(p *tableInternal, r, hoverRow int) string {
	cells := make([]string, 0, len(p.cols))
	for i, value := range p.rows[r] {
		if i >= len(p.cols) || p.cols[i].Width <= 0 {
			continue
		}
		width := p.cols[i].Width
		style := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true)
		renderedCell := p.styles.Cell.Render(style.Render(ansi.Truncate(value, width, "…")))
		cells = append(cells, renderedCell)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	if r == p.cursor {
		return p.styles.Selected.Render(row)
	}
	if r == hoverRow {
		return hoverRowStyle.Render(row)
	}
	return row
}

func setTableWindow(p *tableInternal, start, end, yOffset, hoverRow int) {
	p.start = clampInt(start, 0, len(p.rows))
	p.end = clampInt(end, p.start, len(p.rows))

	renderedRows := make([]string, 0, p.end-p.start)
	for i := p.start; i < p.end; i++ {
		renderedRows = append(renderedRows, renderTableRow(p, i, hoverRow))
	}
	p.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, renderedRows...))

	maxOffset := max((p.end-p.start)-p.viewport.Height(), 0)
	p.viewport.SetYOffset(clampInt(yOffset, 0, maxOffset))
}

func tableWindowForTop(p *tableInternal, top int) (start, end, yOffset int) {
	height := p.viewport.Height()
	windowSize := height * 2
	start = clampInt(top, 0, max(len(p.rows)-windowSize, 0))
	end = min(start+windowSize, len(p.rows))
	return start, end, top - start
}

// tableSetStyles updates row styling without calling table.SetStyles, whose
// UpdateViewport call would re-anchor the window around the cursor.
func tableSetStyles(h *hoverTable, styles table.Styles) {
	p := tableInternals(&h.Model)
	yOffset := p.viewport.YOffset()
	p.styles = styles
	setTableWindow(p, p.start, p.end, yOffset, h.hoverRow)
}

// tableSelectAt sets the cursor to absRow while keeping the row currently at
// the top of the visible area pinned in place. SetCursor alone re-anchors the
// rendered window around the new cursor, which visually shifts the display —
// callers want a click to just highlight, not scroll.
func tableSelectAt(h *hoverTable, absRow int) {
	tableMoveTo(h, absRow)
}

func tableMoveTo(h *hoverTable, absRow int) {
	p := tableInternals(&h.Model)
	height := p.viewport.Height()
	if height <= 0 || len(p.rows) == 0 {
		return
	}

	oldTop := firstVisibleRow(h.Model)
	newCursor := clampInt(absRow, 0, len(p.rows)-1)
	newTop := oldTop
	if newCursor < newTop {
		newTop = newCursor
	} else if newCursor >= newTop+height {
		newTop = newCursor - height + 1
	}
	newTop = clampInt(newTop, 0, max(len(p.rows)-height, 0))

	p.cursor = newCursor
	start, end, yOffset := tableWindowForTop(p, newTop)
	setTableWindow(p, start, end, yOffset, h.hoverRow)
}

func tableMoveBy(h *hoverTable, delta int) {
	p := tableInternals(&h.Model)
	tableMoveTo(h, p.cursor+delta)
}

func tableHandleKey(h *hoverTable, msg tea.KeyPressMsg) bool {
	p := tableInternals(&h.Model)
	switch {
	case key.Matches(msg, p.KeyMap.LineUp):
		tableMoveBy(h, -1)
	case key.Matches(msg, p.KeyMap.LineDown):
		tableMoveBy(h, 1)
	case key.Matches(msg, p.KeyMap.PageUp):
		tableMoveBy(h, -p.viewport.Height())
	case key.Matches(msg, p.KeyMap.PageDown):
		tableMoveBy(h, p.viewport.Height())
	case key.Matches(msg, p.KeyMap.HalfPageUp):
		tableMoveBy(h, -max(p.viewport.Height()/2, 1))
	case key.Matches(msg, p.KeyMap.HalfPageDown):
		tableMoveBy(h, max(p.viewport.Height()/2, 1))
	case key.Matches(msg, p.KeyMap.GotoTop):
		tableMoveTo(h, 0)
	case key.Matches(msg, p.KeyMap.GotoBottom):
		tableMoveTo(h, len(p.rows)-1)
	default:
		return false
	}
	return true
}

// tableScrollBy scrolls the visible window by delta rows without changing the
// selected row. This lets users inspect rows around the current selection
// without implicitly selecting another package.
func tableScrollBy(h *hoverTable, delta int) {
	tableScrollTo(h, firstVisibleRow(h.Model)+delta)
}

func tableScrollTo(h *hoverTable, top int) {
	p := tableInternals(&h.Model)
	height := p.viewport.Height()
	if height <= 0 {
		return
	}
	if len(p.rows) == 0 {
		setTableWindow(p, 0, 0, 0, h.hoverRow)
		return
	}

	maxTop := max(len(p.rows)-height, 0)
	top = clampInt(top, 0, maxTop)

	// The upstream table only renders cursor +/- height rows. If the cursor is
	// near the top, that leaves no off-screen rows for the viewport to scroll
	// into, so rebuild the rendered window around the desired top row.
	start, end, yOffset := tableWindowForTop(p, top)
	setTableWindow(p, start, end, yOffset, h.hoverRow)
}
