package tui

import "testing"

func TestComputeLayoutUsesMeasuredRegions(t *testing.T) {
	layout := computeLayout(121, 40, 2)

	if layout.title != (rect{x: 0, y: 0, w: 121, h: 1}) {
		t.Fatalf("title rect=%+v", layout.title)
	}
	if layout.help != (rect{x: 0, y: 38, w: 121, h: 2}) {
		t.Fatalf("help rect=%+v", layout.help)
	}
	if layout.leftPane.w+layout.rightPane.w != layout.screen.w {
		t.Fatalf("pane widths=%d+%d want %d", layout.leftPane.w, layout.rightPane.w, layout.screen.w)
	}
	if layout.leftTable.w+verticalScrollbarWidth != layout.leftPane.w {
		t.Fatalf("left table width=%d scrollbar=%d pane=%d", layout.leftTable.w, verticalScrollbarWidth, layout.leftPane.w)
	}
	if layout.leftContent.w != layout.leftPane.w || layout.leftContent.h != layout.leftTable.h {
		t.Fatalf("left content rect=%+v left pane=%+v left table=%+v", layout.leftContent, layout.leftPane, layout.leftTable)
	}
	if layout.rightDetail.w+verticalScrollbarWidth != layout.rightPane.w {
		t.Fatalf("right detail width=%d scrollbar=%d pane=%d", layout.rightDetail.w, verticalScrollbarWidth, layout.rightPane.w)
	}
	if layout.rightContent.w != layout.rightPane.w || layout.rightContent.h != layout.rightDetail.h {
		t.Fatalf("right content rect=%+v right pane=%+v right detail=%+v", layout.rightContent, layout.rightPane, layout.rightDetail)
	}
	if layout.leftData.y != layout.leftTable.y+tableHeaderHeight {
		t.Fatalf("left data y=%d want %d", layout.leftData.y, layout.leftTable.y+tableHeaderHeight)
	}
	if layout.leftData.h != layout.leftTable.h-tableHeaderHeight {
		t.Fatalf("left data height=%d want %d", layout.leftData.h, layout.leftTable.h-tableHeaderHeight)
	}
	if layout.leftScrollbar != (rect{x: layout.leftTable.x + layout.leftTable.w, y: layout.leftData.y, w: verticalScrollbarWidth, h: layout.leftData.h}) {
		t.Fatalf("left scrollbar rect=%+v", layout.leftScrollbar)
	}
	if layout.rightScrollbar != (rect{x: layout.rightDetail.x + layout.rightDetail.w, y: layout.rightDetail.y, w: verticalScrollbarWidth, h: layout.rightDetail.h}) {
		t.Fatalf("right scrollbar rect=%+v", layout.rightScrollbar)
	}
}

func TestRectContains(t *testing.T) {
	r := rect{x: 10, y: 3, w: 5, h: 4}
	if !r.contains(10, 3) || !r.contains(14, 6) {
		t.Fatalf("rect should include top-left and bottom-right inner cells")
	}
	if r.contains(15, 6) || r.contains(14, 7) {
		t.Fatalf("rect should exclude right and bottom edges")
	}
}
