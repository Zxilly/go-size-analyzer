package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func newItemDelegate(keys *ItemKeyMapTyp) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		cur, ok := m.SelectedItem().(*wrapper)
		if !ok {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Confirm):
				children := cur.children()
				if len(children) == 0 {
					return m.NewStatusMessage(statusMessageStyle("Nothing to explore"))
				}

				items := make([]list.Item, 0, len(children))
				for _, c := range children {
					items = append(items, c)
				}

				return m.SetItems(items)
			}
		}

		return nil
	}

	help := []key.Binding{keys.Confirm}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}
