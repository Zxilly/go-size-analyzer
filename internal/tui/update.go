package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func (m *viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			// The user pressed up
		case key.Matches(msg, DefaultKeyMap.Down):
			// The user pressed down
		case key.Matches(msg, DefaultKeyMap.Confirm):
			// The user pressed enter
		case key.Matches(msg, DefaultKeyMap.Backward):
			// The user pressed esc
		case key.Matches(msg, DefaultKeyMap.Exit):
			os.Exit(0)
		}
	}
	return m, nil
}

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Confirm  key.Binding
	Backward key.Binding
	Exit     key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view the selected item"),
	),
	Backward: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc/⌫", "go back"),
	),
	Exit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "exit"),
	),
}
