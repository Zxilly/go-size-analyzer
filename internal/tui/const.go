package tui

import "charm.land/lipgloss/v2"

type focusState int

const (
	focusedDetail focusState = iota
	focusedMain
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("69")).
	BorderBottom(true).
	BorderTop(true)

const (
	rowWidthSize = 13
)
