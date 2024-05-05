package tui

import (
	"github.com/Zxilly/go-size-analyzer/internal/result"
	tea "github.com/charmbracelet/bubbletea"
)

var _ tea.Model = (*viewModel)(nil)

type viewModel struct {
	result *result.Result
}

func newViewModel(result *result.Result) *viewModel {
	return &viewModel{result: result}
}

func (m *viewModel) Init() tea.Cmd {
	//TODO implement me
	panic("implement me")
}
