package tui

import (
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var _ tea.Model = (*viewModel)(nil)

type viewModel struct {
	items   []list.Item
	current *wrapper // nil means root

	list list.Model
}

func buildRootItems(result *result.Result) []list.Item {
	ret := make([]list.Item, 0)
	for _, p := range result.Packages {
		ret = append(ret, newWrapper(p))
	}
	for _, s := range result.Sections {
		ret = append(ret, newWrapper(s))
	}
	return ret
}

func newViewModel(result *result.Result) *viewModel {
	delegate := newItemDelegate(&ItemKeyMap)
	entryList := list.New(buildRootItems(result), delegate, 0, 0)
	entryList.Title = result.Name
	entryList.Styles.Title = titleStyle
	return &viewModel{
		items:   buildRootItems(result),
		current: nil,
	}
}

func (m *viewModel) Init() tea.Cmd {
	return nil
}
