package tui

import "github.com/charmbracelet/lipgloss"

type focusState int

const (
	focusedDetail focusState = iota
	focusedMain
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("69"))

const (
	rowWidthSize = 13
)
