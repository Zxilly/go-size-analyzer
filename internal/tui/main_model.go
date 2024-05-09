package tui

import (
	"cmp"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
	"os"
	"slices"
)

var _ tea.Model = (*mainModel)(nil)

type mainModel struct {
	width  int
	height int

	baseItems wrappers

	current *wrapper // nil means root

	fileName string

	leftTable   table.Model
	rightDetail detailModel
	help        help.Model

	focus focusState
}

func (m mainModel) currentSelection() wrapper {
	var list []wrapper
	if m.current == nil {
		list = m.baseItems
	} else {
		list = m.current.children()
	}
	if m.leftTable.Cursor() < 0 || m.leftTable.Cursor() >= len(list) {
		panic("cursor out of range")
	}
	return list[m.leftTable.Cursor()]
}

func (m mainModel) getKeyMap() help.KeyMap {
	mainKeys := []key.Binding{DefaultKeyMap.Switch, DefaultKeyMap.Exit}
	if m.currentSelection().hasChildren() {
		mainKeys = append(mainKeys, DefaultKeyMap.Enter)
	}
	if m.current != nil {
		mainKeys = append(mainKeys, DefaultKeyMap.Backward)
	}

	ret := DynamicKeyMap{
		Short: mainKeys,
		Long:  [][]key.Binding{mainKeys},
	}

	switch m.focus {
	case focusedMain:
		ret.Short = append(ret.Short, tableKeyMap()...)
		ret.Long = append(ret.Long, tableKeyMap())
	case focusedDetail:
		ret.Short = append(ret.Short, m.rightDetail.KeyMap()...)
		ret.Long = append(ret.Long, m.rightDetail.KeyMap())
	}

	return ret
}

func (m mainModel) nextFocus() focusState {
	switch m.focus {
	case focusedMain:
		return focusedDetail
	case focusedDetail:
		return focusedMain
	default:
		panic("invalid focus state")
	}
}

func buildRootItems(result *result.Result) wrappers {
	ret := make([]wrapper, 0)
	for _, p := range result.Packages {
		ret = append(ret, newWrapper(p))
	}
	for _, s := range result.Sections {
		ret = append(ret, newWrapper(s))
	}

	slices.SortFunc(ret, func(a, b wrapper) int {
		return -cmp.Compare(a.size(), b.size())
	})

	return ret
}

func newMainModel(result *result.Result) mainModel {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		utils.FatalError(err)
	}

	baseItems := buildRootItems(result)

	m := mainModel{
		baseItems: baseItems,
		current:   nil,
		fileName:  result.Name,

		rightDetail: newDetailModel(width-width/2, height-3),
		leftTable: table.New(
			table.WithColumns(getTableColumns(width)),
			table.WithRows(baseItems.ToRows()),
			table.WithFocused(true),
		),
		help: help.New(),

		width:  width,
		height: height,

		focus: focusedMain,
	}

	m.rightDetail.viewPort.SetContent(m.currentSelection().Description())

	m, _ = m.handleWindowSizeEvent(width, height)

	return m
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

func (m mainModel) title() string {
	if m.current == nil {
		return m.fileName
	}
	return m.current.Title()
}
