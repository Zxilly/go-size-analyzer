package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func (m *viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			// The user pressed up
		case key.Matches(msg, DefaultKeyMap.Down):
			// The user pressed down

		case key.Matches(msg, DefaultKeyMap.Exit):
			os.Exit(0)
		}
	}
	return m, nil
}

type KeyMapTyp struct {
	Up       key.Binding
	Down     key.Binding
	Backward key.Binding
	Exit     key.Binding
}

type ItemKeyMapTyp struct {
	Confirm key.Binding
}

var ItemKeyMap = ItemKeyMapTyp{
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "explore the selected item"),
	),
}

var DefaultKeyMap = KeyMapTyp{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
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
