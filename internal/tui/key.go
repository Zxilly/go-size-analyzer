package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
)

type KeyMapTyp struct {
	Switch   key.Binding
	Backward key.Binding
	Enter    key.Binding
	Help     key.Binding
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
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "explore"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
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
