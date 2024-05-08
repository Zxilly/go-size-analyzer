package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
)

type KeyMapTyp struct {
	Switch   key.Binding
	Backward key.Binding
	Exit     key.Binding
}

var DefaultKeyMap = KeyMapTyp{
	Switch: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch focus"),
	),
	Backward: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc/backspace", "go back"),
	),
	Exit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "exit"),
	),
}

var _ help.KeyMap = (*DynamicKeyMap)(nil)

type DynamicKeyMap struct {
	Short []key.Binding
	Long  [][]key.Binding
}

func (d DynamicKeyMap) ShortHelp() []key.Binding {
	return d.Short
}

func (d DynamicKeyMap) FullHelp() [][]key.Binding {
	return d.Long
}

func tableKeyMap() []key.Binding {
	all := table.DefaultKeyMap()
	return []key.Binding{
		all.LineUp,
		all.LineDown,
		all.PageUp,
		all.PageDown,
		all.HalfPageUp,
		all.HalfPageDown,
		all.GotoTop,
		all.GotoBottom,
	}
}
