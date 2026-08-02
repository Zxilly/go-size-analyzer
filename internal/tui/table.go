package tui

import "charm.land/bubbles/v2/table"

func getTableColumnsForTableWidth(tableWidth int) []table.Column {
	styles := table.DefaultStyles()
	cellFrameWidth := styles.Cell.GetHorizontalFrameSize() * 2
	contentWidth := max(tableWidth-cellFrameWidth, 0)
	sizeWidth := min(rowWidthSize, contentWidth)

	return []table.Column{
		{
			Title: "Name",
			Width: max(contentWidth-sizeWidth, 0),
		},
		{
			Title: "Size",
			Width: sizeWidth,
		},
	}
}
