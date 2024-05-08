package tui

import "github.com/charmbracelet/bubbles/table"

func getTableColumns(width int) []table.Column {
	return []table.Column{
		{
			Title: "Name",
			Width: width/2 - rowWidthType - rowWidthSize - 2,
		},
		{
			Title: "Type",
			Width: rowWidthType,
		},
		{
			"Size",
			rowWidthSize,
		},
	}
}
