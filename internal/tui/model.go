package tui

import (
	"github.com/Zxilly/go-size-analyzer/internal/result"
	tea "github.com/charmbracelet/bubbletea"
)

var _ tea.Model = (*viewModel)(nil)

type viewModel struct {
	items   []wrapper
	current *wrapper // nil means root
}

func buildRootItems(result *result.Result) []wrapper {
	ret := make([]wrapper, 0)
	for _, p := range result.Packages {
		ret = append(ret, newWrapper(p))
	}
	for _, s := range result.Sections {
		ret = append(ret, newWrapper(s))
	}
	return ret
}

func newViewModel(result *result.Result) *viewModel {
	return &viewModel{
		items:   buildRootItems(result),
		current: nil,
	}
}

func (m *viewModel) Init() tea.Cmd {
	return nil
}
