package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func testResultWithPackages(prefix string, count int) *result.Result {
	pkgs := make(entity.PackageMap, count)
	for i := range count {
		pkg := entity.NewPackage()
		pkg.Name = fmt.Sprintf("%s-%02d", prefix, i)
		pkg.Type = entity.PackageTypeStd
		pkg.Size = uint64(count - i)
		pkgs[pkg.Name] = pkg
	}
	return &result.Result{Name: prefix + ".bin", Packages: pkgs}
}

func testResultWithLargePackage(fileCount int) *result.Result {
	pkg := entity.NewPackage()
	pkg.Name = "large"
	pkg.Type = entity.PackageTypeStd
	pkg.Size = uint64(fileCount)
	for i := range fileCount {
		pkg.Files = append(pkg.Files, &entity.File{
			FilePath: fmt.Sprintf("file-%03d.go", i),
			PkgName:  pkg.Name,
		})
	}
	return &result.Result{
		Name:     "large.bin",
		Packages: entity.PackageMap{"large": pkg},
	}
}

func testResultWithScrollableParent(siblingCount int) *result.Result {
	parent := entity.NewPackage()
	parent.Name = "parent"
	parent.Type = entity.PackageTypeStd
	parent.Size = uint64(siblingCount + 100)

	child := entity.NewPackage()
	child.Name = "child"
	child.Type = entity.PackageTypeStd
	child.Size = 1
	parent.SubPackages[child.Name] = child

	pkgs := make(entity.PackageMap, siblingCount+1)
	pkgs[parent.Name] = parent
	for i := range siblingCount {
		pkg := entity.NewPackage()
		pkg.Name = fmt.Sprintf("sibling-%02d", i)
		pkg.Type = entity.PackageTypeStd
		pkg.Size = uint64(siblingCount - i)
		pkgs[pkg.Name] = pkg
	}

	return &result.Result{
		Name:     "parent.bin",
		Packages: pkgs,
	}
}

func TestNewMainModelUsesChildSelectionStyle(t *testing.T) {
	parent := entity.NewPackage()
	parent.Name = "parent"
	parent.Type = entity.PackageTypeStd
	parent.Size = 100
	child := entity.NewPackage()
	child.Name = "child"
	child.Type = entity.PackageTypeStd
	child.Size = 10
	parent.SubPackages["child"] = child

	m := newMainModel(&result.Result{
		Name:     "test.bin",
		Packages: entity.PackageMap{"parent": parent},
	}, 120, 40)

	if !m.currentSelection().hasChildren() {
		t.Fatal("test setup expected initial selection to have children")
	}

	p := (*tableInternal)(unsafe.Pointer(&m.leftTable.Model))
	got := p.styles.Selected.GetForeground()
	if !reflect.DeepEqual(got, colorSelected) {
		t.Fatalf("initial selected foreground=%#v want %#v", got, colorSelected)
	}
}

func TestNewMainModelDoesNotReuseRootItemsAcrossResults(t *testing.T) {
	first := entity.NewPackage()
	first.Name = "first"
	first.Type = entity.PackageTypeStd
	first.Size = 100

	second := entity.NewPackage()
	second.Name = "second"
	second.Type = entity.PackageTypeStd
	second.Size = 100

	_ = newMainModel(&result.Result{
		Name:     "first.bin",
		Packages: entity.PackageMap{"first": first},
	}, 120, 40)
	m := newMainModel(&result.Result{
		Name:     "second.bin",
		Packages: entity.PackageMap{"second": second},
	}, 120, 40)

	if got := m.currentSelection().Title(); got != "second" {
		t.Fatalf("second model initial selection=%q want second", got)
	}
}

func TestMouseWheelOnlyRoutesInsideContent(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{X: 1, Y: m.layout.title.y, Button: tea.MouseWheelDown})
	if got := firstVisibleRow(m.leftTable.Model); got != 0 {
		t.Fatalf("firstVisibleRow after title wheel=%d want 0", got)
	}

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{X: 1, Y: m.layout.help.y, Button: tea.MouseWheelDown})
	if got := firstVisibleRow(m.leftTable.Model); got != 0 {
		t.Fatalf("firstVisibleRow after help wheel=%d want 0", got)
	}

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{X: 1, Y: m.layout.leftContent.y, Button: tea.MouseWheelDown})
	if got := firstVisibleRow(m.leftTable.Model); got == 0 {
		t.Fatalf("firstVisibleRow after content wheel=%d want non-zero", got)
	}
}

func TestEnterOnLeafSelectionDoesNotChangeFocus(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 120, 40)
	if m.currentSelection().hasChildren() {
		t.Fatal("test setup expected leaf selection")
	}

	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)

	if m.focus != focusedMain {
		t.Fatalf("focus after leaf enter=%d want focusedMain", m.focus)
	}
	if got := m.currentSelection().Title(); got != "pkg-00" {
		t.Fatalf("selection after leaf enter=%q want pkg-00", got)
	}
}

func TestNonLeftMouseClickDoesNotSelectOrDrag(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	data := m.layout.leftData

	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X:      data.x,
		Y:      data.y + 3,
		Button: tea.MouseRight,
	})
	if got := m.leftTable.Cursor(); got != 0 {
		t.Fatalf("cursor after right-click select=%d want 0", got)
	}
	if got := m.currentSelection().Title(); got != "pkg-00" {
		t.Fatalf("selection after right-click=%q want pkg-00", got)
	}

	bar := m.layout.leftScrollbar
	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X:      bar.x,
		Y:      bar.y,
		Button: tea.MouseMiddle,
	})
	if m.drag.target != scrollbarDragNone {
		t.Fatalf("drag target after non-left scrollbar click=%d want none", m.drag.target)
	}
}

func TestLeftScrollbarDragScrollsTable(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	bar := m.layout.leftScrollbar

	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X:      bar.x,
		Y:      bar.y,
		Button: tea.MouseLeft,
	})
	if m.drag.target != scrollbarDragLeft {
		t.Fatalf("drag target=%d want left scrollbar", m.drag.target)
	}

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{
		X:      bar.x,
		Y:      bar.y + bar.h - 1,
		Button: tea.MouseLeft,
	})
	if got := firstVisibleRow(m.leftTable.Model); got == 0 {
		t.Fatalf("firstVisibleRow after dragging left scrollbar=%d want non-zero", got)
	}
	cursor := m.leftTable.Cursor()
	top := firstVisibleRow(m.leftTable.Model)
	if cursor >= top && cursor < top+m.leftTable.Height() {
		t.Fatalf("cursor=%d unexpectedly visible in [%d, %d)", cursor, top, top+m.leftTable.Height())
	}
	if got := m.currentSelection().Title(); got != "pkg-00" {
		t.Fatalf("selection after dragging left scrollbar=%q want pkg-00", got)
	}

	m, _ = handleMouseReleaseEvent(m, tea.MouseReleaseMsg{X: bar.x, Y: bar.y + bar.h - 1})
	if m.drag.target != scrollbarDragNone {
		t.Fatalf("drag target after release=%d want none", m.drag.target)
	}
}

func TestWindowResizePreservesLeftScrollWithOffscreenSelection(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	tableScrollBy(&m.leftTable, 20)
	top := firstVisibleRow(m.leftTable.Model)

	m, _ = handleWindowSizeEvent(m, 100, 35)

	if got := firstVisibleRow(m.leftTable.Model); got != top {
		t.Fatalf("firstVisibleRow after resize=%d want %d", got, top)
	}
	if got := m.currentSelection().Title(); got != "pkg-00" {
		t.Fatalf("selection after resize=%q want pkg-00", got)
	}
}

func TestDoubleClickOnRowEntersChildren(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)
	if !m.currentSelection().hasChildren() {
		t.Fatal("test setup expected initial selection to have children")
	}
	parentTitle := m.currentSelection().Title()

	data := m.layout.leftData
	click := tea.MouseClickMsg{X: data.x, Y: data.y, Button: tea.MouseLeft}

	m, _ = handleMouseClickEvent(m, click)
	if m.current != nil {
		t.Fatalf("entered children after single click; current=%q", m.current.Title())
	}

	m, _ = handleMouseClickEvent(m, click)
	if m.current == nil {
		t.Fatal("did not enter children after double click")
	}
	if got := m.current.Title(); got != parentTitle {
		t.Fatalf("entered=%q want %q", got, parentTitle)
	}
}

func TestSlowSecondClickDoesNotEnterChildren(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)

	current := time.Unix(0, 0)
	restore := nowFunc
	nowFunc = func() time.Time { return current }
	defer func() { nowFunc = restore }()

	data := m.layout.leftData
	click := tea.MouseClickMsg{X: data.x, Y: data.y, Button: tea.MouseLeft}

	m, _ = handleMouseClickEvent(m, click)
	current = current.Add(doubleClickThreshold + time.Millisecond)
	m, _ = handleMouseClickEvent(m, click)

	if m.current != nil {
		t.Fatalf("entered children after slow second click; current=%q", m.current.Title())
	}
}

func TestMouseMotionUpdatesHoverRow(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	if m.leftTable.hoverRow != -1 {
		t.Fatalf("initial hoverRow=%d want -1", m.leftTable.hoverRow)
	}

	data := m.layout.leftData
	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y + 4})
	if m.leftTable.hoverRow != 4 {
		t.Fatalf("hoverRow after motion over row 4=%d want 4", m.leftTable.hoverRow)
	}

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y - 1})
	if m.leftTable.hoverRow != -1 {
		t.Fatalf("hoverRow after motion outside data=%d want -1", m.leftTable.hoverRow)
	}
}

func TestMouseMotionDuringDragIgnoresHover(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	bar := m.layout.leftScrollbar

	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{X: bar.x, Y: bar.y, Button: tea.MouseLeft})
	if m.drag.target != scrollbarDragLeft {
		t.Fatalf("drag target after scrollbar click=%d want left", m.drag.target)
	}

	data := m.layout.leftData
	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y + 3, Button: tea.MouseLeft})
	if m.leftTable.hoverRow != -1 {
		t.Fatalf("hoverRow during drag=%d want -1", m.leftTable.hoverRow)
	}
}

func TestWheelScrollRecomputesHoverRow(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 80), 120, 40)
	data := m.layout.leftData

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y + 3})
	if m.leftTable.hoverRow != 3 {
		t.Fatalf("hoverRow before wheel=%d want 3", m.leftTable.hoverRow)
	}

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{
		X: data.x + 2, Y: data.y + 3, Button: tea.MouseWheelDown,
	})
	want := firstVisibleRow(m.leftTable.Model) + 3
	if m.leftTable.hoverRow != want {
		t.Fatalf("hoverRow after wheel down=%d want %d", m.leftTable.hoverRow, want)
	}
}

func TestWheelScrollOverEmptyAreaClearsHoverRow(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 120, 40)
	data := m.layout.leftData

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y})
	if m.leftTable.hoverRow != 0 {
		t.Fatalf("hoverRow before wheel=%d want 0", m.leftTable.hoverRow)
	}

	// Wheel inside leftContent but at a Y past the last row — recompute
	// should land outside any real row and clear the hover.
	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{
		X: data.x + 2, Y: data.y + 10, Button: tea.MouseWheelDown,
	})
	if m.leftTable.hoverRow != -1 {
		t.Fatalf("hoverRow over empty area after wheel=%d want -1", m.leftTable.hoverRow)
	}
}

func TestRightClickGoesBack(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)
	data := m.layout.leftData

	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	if m.current == nil {
		t.Fatal("test setup expected to be inside a child level after Enter")
	}

	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X: data.x, Y: data.y + 1, Button: tea.MouseRight,
	})
	if m.current != nil {
		t.Fatalf("current after right-click=%q want nil (back to root)", m.current.Title())
	}
}

func TestRightClickOutsideLeftPaneIsIgnored(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)
	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	parentTitle := m.current.Title()

	right := m.layout.rightContent
	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X: right.x, Y: right.y, Button: tea.MouseRight,
	})
	if m.current == nil || m.current.Title() != parentTitle {
		t.Fatalf("right-click on right pane should not navigate; current=%v", m.current)
	}
}

func TestEnterClearsHoverRow(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)
	data := m.layout.leftData

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{X: data.x + 2, Y: data.y})
	if m.leftTable.hoverRow != 0 {
		t.Fatalf("hoverRow before enter=%d want 0", m.leftTable.hoverRow)
	}

	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	if m.leftTable.hoverRow != -1 {
		t.Fatalf("hoverRow after enter=%d want -1", m.leftTable.hoverRow)
	}
}

func TestHelpModeSwitchesWithInputKind(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 120, 40)
	if m.helpMode != helpModeKeyboard {
		t.Fatalf("initial helpMode=%d want keyboard", m.helpMode)
	}

	data := m.layout.leftData
	next, _ := m.Update(tea.MouseClickMsg{X: data.x, Y: data.y, Button: tea.MouseLeft})
	m = next.(mainModel)
	if m.helpMode != helpModeMouse {
		t.Fatalf("helpMode after click=%d want mouse", m.helpMode)
	}

	next, _ = m.Update(tea.MouseMotionMsg{X: data.x + 4, Y: data.y + 1})
	m = next.(mainModel)
	if m.helpMode != helpModeMouse {
		t.Fatalf("helpMode after motion=%d want mouse (motion must not flip)", m.helpMode)
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(mainModel)
	if m.helpMode != helpModeKeyboard {
		t.Fatalf("helpMode after key=%d want keyboard", m.helpMode)
	}
}

func openFullHelp(t *testing.T, m mainModel) mainModel {
	t.Helper()
	next, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = next.(mainModel)
	if !m.help.ShowAll {
		t.Fatalf("expected full help open after ?")
	}
	return m
}

func TestFullHelpTogglesByQuestionMark(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 120, 40)
	m = openFullHelp(t, m)
	next, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = next.(mainModel)
	if m.help.ShowAll {
		t.Fatal("full help should close on second ?")
	}
}

func TestFullHelpRendersOnlyInBottomHelpBar(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 260, 40)
	m = openFullHelp(t, m)

	helpBar := ansi.Strip(m.renderHelpBar())
	if !strings.Contains(helpBar, "toggle help") ||
		!strings.Contains(helpBar, "go to end") ||
		!strings.Contains(helpBar, "left click") {
		t.Fatalf("bottom full help should include expanded keyboard and mouse bindings:\n%s", helpBar)
	}

	content := ansi.Strip(m.renderContent())
	if strings.Contains(content, "╭") || strings.Contains(content, "Close:") {
		t.Fatalf("full help should not render an overlay panel:\n%s", content)
	}
}

func TestFullHelpDoesNotCloseByEsc(t *testing.T) {
	m := newMainModel(testResultWithPackages("pkg", 3), 120, 40)
	m = openFullHelp(t, m)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(mainModel)
	if !m.help.ShowAll {
		t.Fatal("full help should stay open on Esc; only ? toggles it")
	}
}

func TestFullHelpDoesNotSwallowInput(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(0), 120, 40)
	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	if m.current == nil {
		t.Fatal("test setup expected to be inside a child level after Enter")
	}
	m = openFullHelp(t, m)

	data := m.layout.leftData
	next, _ = m.Update(tea.MouseClickMsg{
		X:      data.x,
		Y:      data.y + 1,
		Button: tea.MouseRight,
	})
	m = next.(mainModel)
	if m.current != nil {
		t.Fatalf("right-click while full help is open should go back; current=%q", m.current.Title())
	}
	if !m.help.ShowAll {
		t.Fatal("right-click should not close bottom full help")
	}

	focusBefore := m.focus
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(mainModel)
	if m.focus == focusBefore {
		t.Fatal("Tab while full help is open should still switch focus")
	}
	if !m.help.ShowAll {
		t.Fatal("Tab should not close bottom full help")
	}
}

func TestBackRestoresParentScrollWithOffscreenSelection(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(80), 120, 40)
	tableScrollBy(&m.leftTable, 20)
	top := firstVisibleRow(m.leftTable.Model)

	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	if got := m.title(); got != "parent" {
		t.Fatalf("title after enter=%q want parent", got)
	}

	next, _ = handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = next.(mainModel)
	if got := firstVisibleRow(m.leftTable.Model); got != top {
		t.Fatalf("firstVisibleRow after back=%d want %d", got, top)
	}
	if got := m.currentSelection().Title(); got != "parent" {
		t.Fatalf("selection after back=%q want parent", got)
	}
}

func TestRightScrollbarDragScrollsDetail(t *testing.T) {
	m := newMainModel(testResultWithLargePackage(200), 120, 40)
	bar := m.layout.rightScrollbar

	m, _ = handleMouseClickEvent(m, tea.MouseClickMsg{
		X:      bar.x,
		Y:      bar.y,
		Button: tea.MouseLeft,
	})
	if m.drag.target != scrollbarDragRight {
		t.Fatalf("drag target=%d want right scrollbar", m.drag.target)
	}

	m, _ = handleMouseMotionEvent(m, tea.MouseMotionMsg{
		X:      bar.x,
		Y:      bar.y + bar.h - 1,
		Button: tea.MouseLeft,
	})
	if got := m.rightDetail.viewPort.YOffset(); got == 0 {
		t.Fatalf("detail YOffset after dragging right scrollbar=%d want non-zero", got)
	}
}
