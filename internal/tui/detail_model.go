package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type detailModel struct {
	viewPort viewport.Model
}

func newDetailModel(width, height int) detailModel {
	return detailModel{
		viewPort: viewport.New(width, height),
	}
}

func (d detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd
	d.viewPort, cmd = d.viewPort.Update(msg)
	return d, cmd
}

func (d detailModel) View() string {
	return d.viewPort.View()
}

func (d detailModel) KeyMap() []key.Binding {
	km := d.viewPort.KeyMap
	return []key.Binding{
		km.Up,
		km.Down,
		km.PageUp,
		km.PageDown,
		km.HalfPageUp,
		km.HalfPageDown,
	}
}
