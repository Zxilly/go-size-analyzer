package tui

import "github.com/Zxilly/go-size-analyzer/internal/tui/table"

func getTableColumns(width int, frame int) []table.Column {
	return []table.Column{
		{
			Title: "Name",
			Width: width/2 - rowWidthSize - frame - 7, // fixme: why 7 is ok here?
		},
		{
			"Size",
			rowWidthSize,
		},
	}
}
