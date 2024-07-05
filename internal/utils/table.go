package utils

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func GetTableStyle() table.Style {
	ret := table.StyleLight
	ret.Format.Footer = text.FormatDefault

	return ret
}
