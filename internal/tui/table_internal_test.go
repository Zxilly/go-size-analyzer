package tui

import (
	"fmt"
	"testing"
	"unsafe"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func newTestTable(rowCount, height int) table.Model {
	rows := make([]table.Row, rowCount)
	for i := range rows {
		rows[i] = table.Row{fmt.Sprintf("r%d", i), "x"}
	}
	return table.New(
		table.WithColumns([]table.Column{{Title: "A", Width: 8}, {Title: "B", Width: 1}}),
		table.WithRows(rows),
		table.WithHeight(height),
	)
}

// TestTableInternalLayout guards the unsafe.Pointer cast in table_internal.go.
// Upstream bubbles can change Model's private layout on upgrade — that would
// silently corrupt firstVisibleRow without this canary failing.
func TestTableInternalLayout(t *testing.T) {
	tbl := newTestTable(50, 10)
	if got, want := unsafe.Sizeof(tbl), unsafe.Sizeof(tableInternal{}); got != want {
		t.Fatalf("table.Model size=%d tableInternal size=%d: bubbles layout drift?", got, want)
	}
	if got := firstVisibleRow(tbl); got != 0 {
		t.Fatalf("initial firstVisibleRow=%d want 0", got)
	}
	// MoveDown maintains the cursor-visible invariant (unlike SetCursor).
	tbl.MoveDown(20)
	cursor := tbl.Cursor()
	got := firstVisibleRow(tbl)
	height := tbl.Height()
	if cursor < got || cursor >= got+height {
		t.Fatalf("after MoveDown(20): cursor=%d not in window [%d, %d): bubbles layout drift?", cursor, got, got+height)
	}
}

func TestTableScrollByPreservesCursorOutsideView(t *testing.T) {
	tbl := newTestTable(50, 10)

	tableScrollBy(&tbl, 3)

	if got := firstVisibleRow(tbl); got != 3 {
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

	if got := firstVisibleRow(tbl); got != 3 {
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
	if got := firstVisibleRow(tbl); got != wantBottom {
		t.Fatalf("firstVisibleRow after large scroll down=%d want %d", got, wantBottom)
	}

	tableScrollBy(&tbl, -1000)
	if got := firstVisibleRow(tbl); got != 0 {
		t.Fatalf("firstVisibleRow after large scroll up=%d want 0", got)
	}
}

func TestTableSelectAtPreservesVisibleTop(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	top := firstVisibleRow(tbl)

	tableSelectAt(&tbl, top+2)

	if got := tbl.Cursor(); got != top+2 {
		t.Fatalf("cursor after select=%d want %d", got, top+2)
	}
	if got := firstVisibleRow(tbl); got != top {
		t.Fatalf("firstVisibleRow after select=%d want %d", got, top)
	}
}

func TestTableMoveByPreservesTopWhenCursorRemainsVisible(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl)

	tableMoveBy(&tbl, 1)

	if got := tbl.Cursor(); got != 9 {
		t.Fatalf("cursor after move down=%d want 9", got)
	}
	if got := firstVisibleRow(tbl); got != top {
		t.Fatalf("firstVisibleRow after visible move down=%d want %d", got, top)
	}

	tableMoveBy(&tbl, -1)
	if got := tbl.Cursor(); got != 8 {
		t.Fatalf("cursor after move up=%d want 8", got)
	}
	if got := firstVisibleRow(tbl); got != top {
		t.Fatalf("firstVisibleRow after visible move up=%d want %d", got, top)
	}
}

func TestTableHandleKeyPreservesTopWhenCursorRemainsVisible(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl)

	handled := tableHandleKey(&tbl, tea.KeyPressMsg{Code: tea.KeyDown})

	if !handled {
		t.Fatal("expected down key to be handled")
	}
	if got := tbl.Cursor(); got != 9 {
		t.Fatalf("cursor after down key=%d want 9", got)
	}
	if got := firstVisibleRow(tbl); got != top {
		t.Fatalf("firstVisibleRow after down key=%d want %d", got, top)
	}
}

func TestTableMoveByScrollsOnlyWhenCursorLeavesView(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	top := firstVisibleRow(tbl)
	tableSelectAt(&tbl, top+tbl.Height()-1)
	top = firstVisibleRow(tbl)
	cursor := tbl.Cursor()

	tableMoveBy(&tbl, 1)

	if got := tbl.Cursor(); got != cursor+1 {
		t.Fatalf("cursor after leaving view=%d want %d", got, cursor+1)
	}
	if got := firstVisibleRow(tbl); got != top+1 {
		t.Fatalf("firstVisibleRow after leaving view=%d want %d", got, top+1)
	}
}

func TestTableSetStylesPreservesTop(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 6)
	tableSelectAt(&tbl, 8)
	top := firstVisibleRow(tbl)

	styles := table.DefaultStyles()
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("36"))
	tableSetStyles(&tbl, styles)

	if got := firstVisibleRow(tbl); got != top {
		t.Fatalf("firstVisibleRow after style update=%d want %d", got, top)
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
	p := (*tableInternal)(unsafe.Pointer(&tbl))
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
	if got := firstVisibleRow(tbl); got != wantCursor {
		t.Fatalf("firstVisibleRow after page down=%d want %d", got, wantCursor)
	}
	if got := tbl.Cursor(); got != wantCursor {
		t.Fatalf("cursor after page down=%d want %d", got, wantCursor)
	}
}
