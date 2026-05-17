package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCalculateScrollbar(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		visible     int
		offset      int
		height      int
		wantOffset  int
		wantLength  int
		wantVisible bool
	}{
		{
			name:        "content fits",
			total:       10,
			visible:     10,
			height:      8,
			wantVisible: false,
		},
		{
			name:        "top",
			total:       100,
			visible:     10,
			height:      10,
			wantOffset:  0,
			wantLength:  1,
			wantVisible: true,
		},
		{
			name:        "middle",
			total:       100,
			visible:     10,
			offset:      45,
			height:      10,
			wantOffset:  5,
			wantLength:  1,
			wantVisible: true,
		},
		{
			name:        "bottom",
			total:       100,
			visible:     10,
			offset:      90,
			height:      10,
			wantOffset:  9,
			wantLength:  1,
			wantVisible: true,
		},
		{
			name:        "nearly full content still leaves travel room",
			total:       11,
			visible:     10,
			offset:      1,
			height:      10,
			wantOffset:  1,
			wantLength:  9,
			wantVisible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := calculateScrollbar(tt.total, tt.visible, tt.offset, tt.height)
			if ok != tt.wantVisible {
				t.Fatalf("calculateScrollbar visible=%v want %v", ok, tt.wantVisible)
			}
			if !ok {
				return
			}
			if got.thumbOffset != tt.wantOffset || got.thumbLength != tt.wantLength {
				t.Fatalf("calculateScrollbar=%+v want offset=%d length=%d", got, tt.wantOffset, tt.wantLength)
			}
		})
	}
}

func TestVerticalScrollbarView(t *testing.T) {
	got := ansi.Strip(verticalScrollbarView(100, 10, 90, 10))
	lines := strings.Split(got, "\n")
	if len(lines) != 10 {
		t.Fatalf("scrollbar height=%d want 10", len(lines))
	}
	if lines[9] != scrollbarThumbChar {
		t.Fatalf("last scrollbar line=%q want thumb", lines[9])
	}
}

func TestScrollbarOffsetForY(t *testing.T) {
	metrics := scrollbarMetrics{total: 100, visible: 10, offset: 0, height: 10}

	offset, ok := scrollbarOffsetForY(metrics, 9, 0)

	if !ok {
		t.Fatal("expected y position to map to scroll offset")
	}
	if offset != 90 {
		t.Fatalf("offset=%d want 90", offset)
	}
}

func TestScrollbarGrabOffset(t *testing.T) {
	metrics := scrollbarMetrics{total: 100, visible: 10, offset: 45, height: 10}

	grabOffset, ok := scrollbarGrabOffset(metrics, 5)

	if !ok {
		t.Fatal("expected thumb to be draggable")
	}
	if grabOffset != 0 {
		t.Fatalf("grabOffset=%d want 0", grabOffset)
	}
}

func TestTableScrollbarViewIncludesHeaderSpacer(t *testing.T) {
	tbl := newTestTable(50, 10)
	tableScrollBy(&tbl, 40)

	got := ansi.Strip(tableScrollbarView(tbl))
	lines := strings.Split(got, "\n")
	if len(lines) != tbl.Height()+1 {
		t.Fatalf("table scrollbar height=%d want %d", len(lines), tbl.Height()+1)
	}
	if lines[0] != " " {
		t.Fatalf("header spacer=%q want blank column", lines[0])
	}
	if lines[len(lines)-1] != scrollbarThumbChar {
		t.Fatalf("bottom scrollbar line=%q want thumb", lines[len(lines)-1])
	}
}
