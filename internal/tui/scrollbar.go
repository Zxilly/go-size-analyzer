package tui

import (
	"math"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

const (
	verticalScrollbarWidth = 1
	scrollbarTrackChar     = "│"
	scrollbarThumbChar     = "█"
)

var (
	scrollbarTrackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	scrollbarThumbStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

type scrollbarState struct {
	thumbOffset int
	thumbLength int
}

type scrollbarMetrics struct {
	total   int
	visible int
	offset  int
	height  int
}

type scrollbarDragTarget int

const (
	scrollbarDragNone scrollbarDragTarget = iota
	scrollbarDragLeft
	scrollbarDragRight
)

type scrollbarDrag struct {
	target     scrollbarDragTarget
	grabOffset int
}

func calculateScrollbar(total, visible, offset, height int) (scrollbarState, bool) {
	if total <= 0 || visible <= 0 || height <= 0 || total <= visible {
		return scrollbarState{}, false
	}

	visible = clampInt(visible, 0, total)
	offset = clampInt(offset, 0, total-visible)

	thumbLength := int(math.Round(float64(visible) / float64(total) * float64(height)))
	thumbLength = clampInt(thumbLength, 1, height)
	if height > 1 {
		thumbLength = min(thumbLength, height-1)
	}

	maxOffset := total - visible
	maxThumbOffset := height - thumbLength
	thumbOffset := 0
	if maxOffset > 0 && maxThumbOffset > 0 {
		thumbOffset = int(math.Round(float64(offset) / float64(maxOffset) * float64(maxThumbOffset)))
	}

	return scrollbarState{
		thumbOffset: clampInt(thumbOffset, 0, maxThumbOffset),
		thumbLength: thumbLength,
	}, true
}

func scrollbarGrabOffset(metrics scrollbarMetrics, yRel int) (int, bool) {
	state, ok := calculateScrollbar(metrics.total, metrics.visible, metrics.offset, metrics.height)
	if !ok {
		return 0, false
	}
	if yRel >= state.thumbOffset && yRel < state.thumbOffset+state.thumbLength {
		return yRel - state.thumbOffset, true
	}
	return state.thumbLength / 2, true
}

func scrollbarOffsetForY(metrics scrollbarMetrics, yRel, grabOffset int) (int, bool) {
	state, ok := calculateScrollbar(metrics.total, metrics.visible, metrics.offset, metrics.height)
	if !ok {
		return 0, false
	}

	maxOffset := metrics.total - metrics.visible
	maxThumbOffset := metrics.height - state.thumbLength
	if maxOffset <= 0 || maxThumbOffset <= 0 {
		return 0, true
	}

	thumbOffset := clampInt(yRel-grabOffset, 0, maxThumbOffset)
	offset := int(math.Round(float64(thumbOffset) / float64(maxThumbOffset) * float64(maxOffset)))
	return clampInt(offset, 0, maxOffset), true
}

func verticalScrollbarView(total, visible, offset, height int) string {
	if height <= 0 {
		return ""
	}

	state, scrollable := calculateScrollbar(total, visible, offset, height)
	lines := make([]string, height)
	for i := range lines {
		if scrollable && i >= state.thumbOffset && i < state.thumbOffset+state.thumbLength {
			lines[i] = scrollbarThumbStyle.Render(scrollbarThumbChar)
			continue
		}
		if scrollable {
			lines[i] = scrollbarTrackStyle.Render(scrollbarTrackChar)
			continue
		}
		lines[i] = " "
	}
	return strings.Join(lines, "\n")
}

func tableScrollbarView(t table.Model) string {
	metrics := tableScrollbarMetrics(t)
	bar := verticalScrollbarView(metrics.total, metrics.visible, metrics.offset, metrics.height)
	if bar == "" {
		return " "
	}
	return " \n" + bar
}

func tableViewWithScrollbar(t table.Model) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, t.View(), tableScrollbarView(t))
}

func tableScrollbarMetrics(t table.Model) scrollbarMetrics {
	dataHeight := t.Height()
	total := len(t.Rows())
	return scrollbarMetrics{
		total:   total,
		visible: min(total, dataHeight),
		offset:  firstVisibleRow(t),
		height:  dataHeight,
	}
}

func detailViewWithScrollbar(d detailModel) string {
	metrics := detailScrollbarMetrics(d)
	bar := verticalScrollbarView(metrics.total, metrics.visible, metrics.offset, metrics.height)
	return lipgloss.JoinHorizontal(lipgloss.Top, d.View(), bar)
}

func detailScrollbarMetrics(d detailModel) scrollbarMetrics {
	height := d.viewPort.Height()
	total := d.viewPort.TotalLineCount()
	return scrollbarMetrics{
		total:   total,
		visible: min(total, height),
		offset:  d.viewPort.YOffset(),
		height:  height,
	}
}
