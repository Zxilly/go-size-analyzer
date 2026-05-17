package tui

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

	p := (*tableInternal)(unsafe.Pointer(&m.leftTable))
	got := p.styles.Selected.GetForeground()
	want := lipgloss.Color("36")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("initial selected foreground=%#v want %#v", got, want)
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
	if got := firstVisibleRow(m.leftTable); got != 0 {
		t.Fatalf("firstVisibleRow after title wheel=%d want 0", got)
	}

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{X: 1, Y: m.layout.help.y, Button: tea.MouseWheelDown})
	if got := firstVisibleRow(m.leftTable); got != 0 {
		t.Fatalf("firstVisibleRow after help wheel=%d want 0", got)
	}

	m, _ = handleMouseWheelEvent(m, tea.MouseWheelMsg{X: 1, Y: m.layout.leftContent.y, Button: tea.MouseWheelDown})
	if got := firstVisibleRow(m.leftTable); got == 0 {
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
	if got := firstVisibleRow(m.leftTable); got == 0 {
		t.Fatalf("firstVisibleRow after dragging left scrollbar=%d want non-zero", got)
	}
	cursor := m.leftTable.Cursor()
	top := firstVisibleRow(m.leftTable)
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
	top := firstVisibleRow(m.leftTable)

	m, _ = handleWindowSizeEvent(m, 100, 35)

	if got := firstVisibleRow(m.leftTable); got != top {
		t.Fatalf("firstVisibleRow after resize=%d want %d", got, top)
	}
	if got := m.currentSelection().Title(); got != "pkg-00" {
		t.Fatalf("selection after resize=%q want pkg-00", got)
	}
}

func TestBackRestoresParentScrollWithOffscreenSelection(t *testing.T) {
	m := newMainModel(testResultWithScrollableParent(80), 120, 40)
	tableScrollBy(&m.leftTable, 20)
	top := firstVisibleRow(m.leftTable)

	next, _ := handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(mainModel)
	if got := m.title(); got != "parent" {
		t.Fatalf("title after enter=%q want parent", got)
	}

	next, _ = handleKeyEvent(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = next.(mainModel)
	if got := firstVisibleRow(m.leftTable); got != top {
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
