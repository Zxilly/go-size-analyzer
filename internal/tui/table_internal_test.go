package tui

import (
	"fmt"
	"testing"
	"unsafe"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

func newTestTable(rowCount, height int) hoverTable {
	rows := make([]table.Row, rowCount)
	for i := range rows {
		rows[i] = table.Row{fmt.Sprintf("r%d", i), "x"}
	}
	return hoverTable{
		Model: table.New(
			table.WithColumns([]table.Column{{Title: "A", Width: 8}, {Title: "B", Width: 1}}),
			table.WithRows(rows),
			table.WithHeight(height),
		),
		hoverRow: -1,
	}
}

// TestTableInternalLayout guards the unsafe.Pointer cast in table_internal.go.
// Upstream bubbles can change Model's private layout on upgrade — that would
// silently corrupt firstVisibleRow without this canary failing.
func TestTableInternalLayout(t *testing.T) {
	tbl := newTestTable(50, 10)
	if got, want := unsafe.Sizeof(tbl.Model), unsafe.Sizeof(tableInternal{}); got != want {
		t.Fatalf("table.Model size=%d tableInternal size=%d: bubbles layout drift?", got, want)
	}
	if got := firstVisibleRow(tbl.Model); got != 0 {
		t.Fatalf("initial firstVisibleRow=%d want 0", got)
	}
	// MoveDown maintains the cursor-visible invariant (unlike SetCursor).
	tbl.MoveDown(20)
	cursor := tbl.Cursor()
	got := firstVisibleRow(tbl.Model)
	height := tbl.Height()
	if cursor < got || cursor >= got+height {
		t.Fatalf("after MoveDown(20): cursor=%d not in window [%d, %d): bubbles layout drift?", cursor, got, got+height)
	}
}

func TestTableScrollByPreservesCursorOutsideView(t *testing.T) {
	tbl := newTestTable(50, 10)

	tableScrollBy(&tbl, 3)

	if got := firstVisibleRow(tbl.Model); got != 3 {
		t.Fatalf("firstVisibleRow after wheel scroll=%d want 3", got)
	}
	if got := tbl.Cursor(); got != 0 {
		t.Fatalf("cursor after wheel scroll=%d want 0", got)
	}
}

func TestTableScrollByPreservesCursorWhenItRemainsVisible(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableSelectAt(&tbl, 8)

	tableScrollBy(&tbl, 3)

	if got := firstVisibleRow(tbl.Model); got != 3 {
		t.Fatalf("firstVisibleRow after wheel scroll=%d want 3", got)
	}
	if got := tbl.Cursor(); got != 8 {
		t.Fatalf("cursor after wheel scroll=%d want 8", got)
	}
}

func TestTableScrollByClampsToBounds(t *testing.T) {
	tbl := newTestTable(50, 10)

	tableScrollBy(&tbl, 1000)

	wantBottom := len(tbl.Rows()) - tbl.Height()
	if got := firstVisibleRow(tbl.Model); got != wantBottom {
		t.Fatalf("firstVisibleRow after large scroll down=%d want %d", got, wantBottom)
	}

	tableScrollBy(&tbl, -1000)
	if got := firstVisibleRow(tbl.Model); got != 0 {
		t.Fatalf("firstVisibleRow after large scroll up=%d want 0", got)
	}
}

func TestTableSelectAtPreservesVisibleTop(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	top := firstVisibleRow(tbl.Model)

	tableSelectAt(&tbl, top+2)

	if got := tbl.Cursor(); got != top+2 {
		t.Fatalf("cursor after select=%d want %d", got, top+2)
	}
	if got := firstVisibleRow(tbl.Model); got != top {
		t.Fatalf("firstVisibleRow after select=%d want %d", got, top)
	}
}

func TestTableMoveByPreservesTopWhenCursorRemainsVisible(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl.Model)

	tableMoveBy(&tbl, 1)

	if got := tbl.Cursor(); got != 9 {
		t.Fatalf("cursor after move down=%d want 9", got)
	}
	if got := firstVisibleRow(tbl.Model); got != top {
		t.Fatalf("firstVisibleRow after visible move down=%d want %d", got, top)
	}

	tableMoveBy(&tbl, -1)
	if got := tbl.Cursor(); got != 8 {
		t.Fatalf("cursor after move up=%d want 8", got)
	}
	if got := firstVisibleRow(tbl.Model); got != top {
		t.Fatalf("firstVisibleRow after visible move up=%d want %d", got, top)
	}
}

func TestTableHandleKeyPreservesTopWhenCursorRemainsVisible(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl.Model)

	handled := tableHandleKey(&tbl, tea.KeyPressMsg{Code: tea.KeyDown})

	if !handled {
		t.Fatal("expected down key to be handled")
	}
	if got := tbl.Cursor(); got != 9 {
		t.Fatalf("cursor after down key=%d want 9", got)
	}
	if got := firstVisibleRow(tbl.Model); got != top {
		t.Fatalf("firstVisibleRow after down key=%d want %d", got, top)
	}
}

func TestTableMoveByScrollsOnlyWhenCursorLeavesView(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	top := firstVisibleRow(tbl.Model)
	tableSelectAt(&tbl, top+tbl.Height()-1)
	top = firstVisibleRow(tbl.Model)
	cursor := tbl.Cursor()

	tableMoveBy(&tbl, 1)

	if got := tbl.Cursor(); got != cursor+1 {
		t.Fatalf("cursor after leaving view=%d want %d", got, cursor+1)
	}
	if got := firstVisibleRow(tbl.Model); got != top+1 {
		t.Fatalf("firstVisibleRow after leaving view=%d want %d", got, top+1)
	}
}

func TestTableSetStylesPreservesTop(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl.Model)

	styles := table.DefaultStyles()
	styles.Selected = styles.Selected.Foreground(colorSelected)
	tableSetStyles(&tbl, styles)

	if got := firstVisibleRow(tbl.Model); got != top {
		t.Fatalf("firstVisibleRow after style update=%d want %d", got, top)
	}
}

func TestRenderTableRowCombinesSelectedAndHoverStyles(t *testing.T) {
	tbl := newTestTable(2, 10)
	p := tableInternals(&tbl.Model)
	p.styles.Selected = p.styles.Selected.Foreground(colorSelected)

	p.cursor = 1
	bareRow := renderTableRow(p, 0, -1)

	p.cursor = 0
	got := renderTableRow(p, 0, 0)
	want := p.styles.Selected.Background(colorHoverBg).Render(bareRow)
	if got != want {
		t.Fatalf("selected hover row style mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestTableHandleKeyIgnoresUnrelatedKey(t *testing.T) {
	tbl := newTestTable(50, 10)

	handled := tableHandleKey(&tbl, tea.KeyPressMsg{Code: 'x'})

	if handled {
		t.Fatal("expected unrelated key not to be handled")
	}
}

func TestTableHandleKeyUsesModelKeyMap(t *testing.T) {
	tbl := newTestTable(50, 10)
	p := (*tableInternal)(unsafe.Pointer(&tbl.Model))
	p.KeyMap.LineDown = key.NewBinding(key.WithKeys("x"))

	handled := tableHandleKey(&tbl, tea.KeyPressMsg{Code: 'x'})

	if !handled {
		t.Fatal("expected custom line-down key to be handled")
	}
	if got := tbl.Cursor(); got != 1 {
		t.Fatalf("cursor after custom key=%d want 1", got)
	}
}

func TestTableHandlePageKeyJumpsToOffscreenCursorAfterWheelScroll(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 20)

	handled := tableHandleKey(&tbl, tea.KeyPressMsg{Code: tea.KeyPgDown})

	if !handled {
		t.Fatal("expected page-down key to be handled")
	}
	wantCursor := tbl.Height()
	if got := firstVisibleRow(tbl.Model); got != wantCursor {
		t.Fatalf("firstVisibleRow after page down=%d want %d", got, wantCursor)
	}
	if got := tbl.Cursor(); got != wantCursor {
		t.Fatalf("cursor after page down=%d want %d", got, wantCursor)
	}
}
