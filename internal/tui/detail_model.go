package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

type detailModel struct {
	display *wrapper
}

func (d detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	//TODO implement me
	return d, nil
}

func (d detailModel) View() string {
	//TODO implement me
	return ""
}

func (d detailModel) KeyMap() help.KeyMap {
	//TODO implement me
	return nil
}
