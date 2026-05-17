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

// mouseBindings reuses key.Binding for display only — the Key field is the
// mouse action label, never matched against any input. help.Model only
// renders Help().Key and Help().Desc, so empty WithKeys is harmless.
//
// mouseAllBindings is ordered so the short bar is a prefix of the full list.
var (
	mouseBindings = struct {
		LeftClick, DoubleClick, RightClick, Wheel, DragScroll key.Binding
	}{
		LeftClick:   key.NewBinding(key.WithHelp("left click", "select / focus")),
		DoubleClick: key.NewBinding(key.WithHelp("double click", "explore")),
		RightClick:  key.NewBinding(key.WithHelp("right click", "go back")),
		Wheel:       key.NewBinding(key.WithHelp("wheel", "scroll")),
		DragScroll:  key.NewBinding(key.WithHelp("drag scrollbar", "scroll viewport")),
	}

	mouseAllBindings = []key.Binding{
		mouseBindings.LeftClick,
		mouseBindings.DoubleClick,
		mouseBindings.RightClick,
		mouseBindings.Wheel,
		mouseBindings.DragScroll,
	}

	mouseShortList = mouseAllBindings[:4]
	mouseFullList  = mouseAllBindings
)

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
