package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func TestMappingExcludesBSSAndClipsFileBackedExtent(t *testing.T) {
	sections := []*entity.Section{{Addr: 100, Size: 80, FileSize: 32, Offset: 500}, {Addr: 200, Size: 32, FileSize: 32, OnlyInMemory: true}, {Addr: 300, Size: 32, FileSize: 32, VirtualSection: true}}
	got := mapRanges(120, 200, sections)
	if len(got) != 1 || got[0].Offset != 520 || got[0].Size != 12 {
		t.Fatalf("mapping: %+v", got)
	}
	if got := mapRanges(^uint64(0)-1, 4, sections); len(got) != 0 {
		t.Fatal("overflow mapped")
	}
}

func TestProjectionKeepsObjectIdentityAndDoesNotSumOverlappingClaims(t *testing.T) {
	fn := &entity.Function{Name: "f", Addr: 100, CodeSize: 20, Source: entity.AddrSourceGoPclntab}
	fn.Init()
	fn.SetPclnRanges([]entity.AddrPos{{Addr: 110, Size: 20, Type: entity.AddrTypeData}})
	p := entity.NewPackage()
	p.Name = "example"
	p.Files = []*entity.File{{FilePath: "main.go", Functions: []*entity.Function{fn}}}
	r := &result.Result{Size: 100, Packages: entity.PackageMap{"example": p}, Sections: []*entity.Section{{Addr: 100, Offset: 10, Size: 60, FileSize: 60}}, Coverage: &entity.FileCoverage{Attributed: 30}}
	rep := project(r, 1)
	if len(rep.Objects) != 1 {
		t.Fatalf("objects: %d", len(rep.Objects))
	}
	o := rep.Objects[0]
	if o.PhysicalBytes != 30 || rep.ObjectPhysicalUnion != 30 {
		t.Fatalf("overlap double counted: %+v", o)
	}
	if o.EvidenceCount != 2 || len(o.Evidence) != 1 {
		t.Fatalf("evidence sample/count: %+v", o)
	}
	fn.Addr = 200
	if functionID("example", fn) != o.ID {
		t.Fatal("identity depends on address")
	}
}

func TestDiffExposesCancellingGrowthAndComponentChanges(t *testing.T) {
	old := &report{Size: 1000, Objects: []*object{{ID: "a", Name: "a", PhysicalBytes: 100, CodeBytes: 100}, {ID: "b", Name: "b", PhysicalBytes: 200}, {ID: "c", Name: "c", PhysicalBytes: 100, CodeBytes: 60, PclnBytes: 40}}}
	next := &report{Size: 1000, Objects: []*object{{ID: "a", Name: "a", PhysicalBytes: 200, CodeBytes: 200}, {ID: "b", Name: "b", PhysicalBytes: 100}, {ID: "c", Name: "c", PhysicalBytes: 100, CodeBytes: 50, PclnBytes: 50}}}
	c := compare(old, next)
	if c.FileDelta != 0 || len(c.Changed) != 3 {
		t.Fatalf("hidden object changes: %+v", c)
	}
	byID := map[string]change{}
	for _, ch := range c.Changed {
		byID[ch.ID] = ch
	}
	if byID["a"].Delta != 100 || byID["b"].Delta != -100 || byID["c"].CodeDelta != -10 {
		t.Fatal(byID)
	}
	self := compare(old, old)
	if len(self.Changed) != 0 || self.UnchangedObjects != 3 {
		t.Fatal("self diff changed")
	}
}

func TestNativeTextSymbolCodeBytesAreAUnion(t *testing.T) {
	p := entity.NewPackage()
	p.Name = "cgo"
	s := entity.NewSymbol("native_function", 100, 20, entity.AddrTypeText)
	for _, source := range []string{entity.AddrSourceSymbol, entity.AddrSourceDwarf} {
		p.AddSymbol(s, &entity.Addr{AddrPos: &entity.AddrPos{Addr: 100, Size: 20, Type: entity.AddrTypeText}, Pkg: p, Symbol: s, SourceType: source, Role: entity.AddrRoleDefinition})
	}
	r := &result.Result{Size: 100, Packages: entity.PackageMap{"cgo": p}, Sections: []*entity.Section{{Addr: 100, Offset: 10, Size: 60, FileSize: 60}}, Coverage: &entity.FileCoverage{Attributed: 20}}
	rep := project(r, 3)
	if len(rep.Objects) != 1 || rep.Objects[0].CodeBytes != 20 || rep.Objects[0].PhysicalBytes != 20 {
		t.Fatalf("native code not measured uniquely: %+v", rep.Objects)
	}
}

func TestDiffWarnsAboutBuildAndCoverageChanges(t *testing.T) {
	a := &report{Manifest: manifest{GoVersion: "go1.23", Settings: map[string]string{"GOARCH": "amd64"}}, Coverage: &entity.FileCoverage{Version: 1}, Analyzers: []string{"symbol"}}
	b := &report{Manifest: manifest{GoVersion: "go1.27", Settings: map[string]string{"GOARCH": "arm64"}}, Coverage: &entity.FileCoverage{Version: 2}}
	c := compare(a, b)
	if len(c.Warnings) != 5 {
		t.Fatalf("warnings: %v", c.Warnings)
	}
}

func TestHTMLTreatsBinaryNamesAsText(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.html")
	r := &report{Coverage: &entity.FileCoverage{}, Objects: []*object{{Name: "<script>alert(1)</script>"}}}
	if err := writeHTML(p, r, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<script>alert(1)</script>") || !strings.Contains(string(b), "&lt;script&gt;") {
		t.Fatal("unescaped binary metadata")
	}
}
