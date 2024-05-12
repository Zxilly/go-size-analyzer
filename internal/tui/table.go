package tui

import "github.com/charmbracelet/bubbles/table"

func getTableColumns(width int) []table.Column {
	return []table.Column{
		{
			Title: "Name",
			Width: width/2 - rowWidthSize - 6, // fixme: why 6 is ok here?
		},
		{
			Title: "Size",
			Width: rowWidthSize,
		},
	}
}
