package tui

import "charm.land/lipgloss/v2"

type rect struct {
	x int
	y int
	w int
	h int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type tuiLayout struct {
	screen rect
	title  rect
	main   rect
	help   rect

	leftPane  rect
	rightPane rect

	leftContent  rect
	rightContent rect

	leftTable   rect
	leftData    rect
	rightDetail rect

	leftScrollbar  rect
	rightScrollbar rect

	helpDialog      rect
	helpDialogClose rect
}

const (
	minTerminalWidth  = 70
	minTerminalHeight = 20
	titleHeight       = 1
	tableHeaderHeight = 1

	helpDialogWidth    = 60
	helpDialogHeight   = 12
	helpDialogCloseLabel = "[ × ]"
)

// helpDialogCloseW is the rendered cell width of helpDialogCloseLabel. `×`
// is ambiguous-width in Unicode; defer to lipgloss instead of hard-coding 5.
var helpDialogCloseW = lipgloss.Width(helpDialogCloseLabel)

func computeLayout(width, height, helpHeight int) tuiLayout {
	helpHeight = clampInt(helpHeight, 0, max(height-titleHeight, 0))

	title := rect{x: 0, y: 0, w: width, h: titleHeight}
	help := rect{x: 0, y: height - helpHeight, w: width, h: helpHeight}
	main := rect{x: 0, y: title.h, w: width, h: max(height-title.h-help.h, 0)}

	leftPaneWidth := width / 2
	leftPane := rect{x: 0, y: main.y, w: leftPaneWidth, h: main.h}
	rightPane := rect{x: leftPane.w, y: main.y, w: max(width-leftPane.w, 0), h: main.h}

	contentY := main.y
	if baseStyle.GetBorderTop() {
		contentY++
	}
	contentHeight := max(main.h-baseStyle.GetVerticalFrameSize(), 0)

	leftContent := rect{
		x: leftPane.x,
		y: contentY,
		w: leftPane.w,
		h: contentHeight,
	}
	rightContent := rect{
		x: rightPane.x,
		y: contentY,
		w: rightPane.w,
		h: contentHeight,
	}

	leftTable := leftContent
	leftTable.w = max(leftContent.w-verticalScrollbarWidth, 0)
	leftData := rect{
		x: leftTable.x,
		y: leftTable.y + tableHeaderHeight,
		w: leftTable.w,
		h: max(leftTable.h-tableHeaderHeight, 0),
	}
	leftScrollbar := rect{
		x: leftTable.x + leftTable.w,
		y: leftData.y,
		w: verticalScrollbarWidth,
		h: leftData.h,
	}
	rightDetail := rightContent
	rightDetail.w = max(rightContent.w-verticalScrollbarWidth, 0)
	rightScrollbar := rect{
		x: rightDetail.x + rightDetail.w,
		y: rightDetail.y,
		w: verticalScrollbarWidth,
		h: rightDetail.h,
	}

	dialogW := min(helpDialogWidth, max(width-4, 0))
	dialogH := min(helpDialogHeight, max(height-4, 0))
	dialog := rect{
		x: (width - dialogW) / 2,
		y: (height - dialogH) / 2,
		w: dialogW,
		h: dialogH,
	}
	dialogClose := rect{
		x: dialog.x + dialog.w - helpDialogCloseW - 1,
		y: dialog.y,
		w: helpDialogCloseW,
		h: 1,
	}

	return tuiLayout{
		screen: rect{x: 0, y: 0, w: width, h: height},
		title:  title,
		main:   main,
		help:   help,

		leftPane:  leftPane,
		rightPane: rightPane,

		leftContent:  leftContent,
		rightContent: rightContent,

		leftTable:   leftTable,
		leftData:    leftData,
		rightDetail: rightDetail,

		leftScrollbar:  leftScrollbar,
		rightScrollbar: rightScrollbar,

		helpDialog:      dialog,
		helpDialogClose: dialogClose,
	}
}
