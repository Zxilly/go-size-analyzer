package tui

import "charm.land/lipgloss/v2"

type focusState int

const (
	focusedDetail focusState = iota
	focusedMain
)

type helpMode int

const (
	helpModeKeyboard helpMode = iota
	helpModeMouse
)

// xterm-256 palette indices used across the TUI. Centralised here so the
// "raw number" only appears once per role and adjustments stay coherent.
var (
	colorBorder         = lipgloss.Color("69")  // focused pane border + scrollbar thumb
	colorBorderDisabled = lipgloss.Color("241") // unfocused pane border
	colorSelected       = lipgloss.Color("36")  // selected row foreground (teal)
	colorHoverBg        = lipgloss.Color("237") // hovered row background
	colorScrollbarBg    = lipgloss.Color("240") // scrollbar track
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(colorBorder).
	BorderBottom(true).
	BorderTop(true)

const (
	rowWidthSize = 13
)
