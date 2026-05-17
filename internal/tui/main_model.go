package tui

import (
	"cmp"
	"slices"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/Zxilly/go-size-analyzer/internal/result"
)

var _ tea.Model = (*mainModel)(nil)

type mainModel struct {
	width  int
	height int
	layout tuiLayout
	drag   scrollbarDrag

	baseItems wrappers

	current *wrapper // nil means root

	fileName string

	leftTable   hoverTable
	rightDetail detailModel
	help        help.Model

	focus    focusState
	helpMode helpMode

	parents []hoverTable

	lastClickAt  time.Time
	lastClickRow int
}

func (m mainModel) currentSelection() *wrapper {
	l := m.currentList()
	if m.leftTable.Cursor() < 0 || m.leftTable.Cursor() >= len(l) {
		panic("cursor out of range")
	}
	return &l[m.leftTable.Cursor()]
}

func (m mainModel) currentList() wrappers {
	if m.current == nil {
		return m.baseItems
	}
	return m.current.children()
}

func (m mainModel) keyboardBindings() ([]key.Binding, [][]key.Binding) {
	mainKeys := []key.Binding{DefaultKeyMap.Switch, DefaultKeyMap.Help, DefaultKeyMap.Exit}
	if m.currentSelection().hasChildren() {
		mainKeys = append(mainKeys, DefaultKeyMap.Enter)
	}
	if m.current != nil {
		mainKeys = append(mainKeys, DefaultKeyMap.Backward)
	}

	var focusKeys []key.Binding
	switch m.focus {
	case focusedMain:
		focusKeys = tableKeyMap()
	case focusedDetail:
		focusKeys = m.rightDetail.KeyMap()
	}

	return append(mainKeys, focusKeys...), [][]key.Binding{mainKeys, focusKeys}
}

func (m mainModel) getKeyMap() help.KeyMap {
	short, long := m.keyboardBindings()
	if m.helpMode == helpModeMouse {
		// Mouse short bar still carries the keyboard essentials so the
		// user never loses sight of how to quit or open help.
		short = append(append([]key.Binding{}, mouseShortList...),
			DefaultKeyMap.Help, DefaultKeyMap.Exit)
	}
	return DynamicKeyMap{Short: short, Long: long}
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

func buildRootItems(r *result.Result) wrappers {
	ret := make([]wrapper, 0)
	for _, p := range r.Packages {
		ret = append(ret, newWrapper(p))
	}
	for _, s := range r.Sections {
		ret = append(ret, newWrapper(s))
	}

	slices.SortFunc(ret, func(a, b wrapper) int {
		return -cmp.Compare(a.size(), b.size())
	})

	return ret
}

func newLeftTable(width int, rows []table.Row) hoverTable {
	return hoverTable{
		Model: table.New(
			table.WithColumns(getTableColumnsForTableWidth(width)),
			table.WithRows(rows),
			table.WithFocused(true),
		),
		hoverRow: -1,
	}
}

func newMainModel(r *result.Result, width, height int) mainModel {
	baseItems := buildRootItems(r)

	m := mainModel{
		baseItems:   baseItems,
		fileName:    r.Name,
		rightDetail: newDetailModel(),
		leftTable:   newLeftTable(width, baseItems.ToRows()),
		help:        help.New(),
		focus:       focusedMain,
	}

	m.width = width
	m.height = height
	m = m.reconcile()

	return m
}

func (mainModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m mainModel) title() string {
	if m.current == nil {
		return m.fileName
	}
	switch {
	case m.current.file != nil:
		return m.current.parent.Title() + "/" + m.current.Title()
	default:
		return m.current.Title()
	}
}
