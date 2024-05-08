package tui

import "github.com/charmbracelet/lipgloss"

type focusState int

const (
	focusedDetail focusState = iota
	focusedChildren
	focusedMain
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240")).
	Inline(true)

const (
	rowWidthType = 10
	rowWidthSize = 13
)
