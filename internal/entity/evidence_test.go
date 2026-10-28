package entity

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type evidenceSummary struct {
	addr, size                  uint64
	pkg, source, role, boundary string
	name                        string
}

func summarizeEvidence(all func(func(*Addr) bool)) []evidenceSummary {
	var out []evidenceSummary
	for addr := range all {
		out = append(out, evidenceSummary{addr.Addr, addr.Size, addrPackageName(addr), addr.SourceType, addr.Role, addr.Boundary, addrEntityName(addr)})
	}
	return out
}

func testEvidenceStore() *Store {
	store := NewStore()
	store.Sections["data"] = &Section{Addr: 100, Size: 1000, FileSize: 1000, ContentType: SectionContentData}
	store.Sections["text"] = &Section{Addr: 2000, Size: 1000, FileSize: 1000, ContentType: SectionContentText}
	store.BuildCache()
	return store
}

func TestAddrSpacePreservesDistinctEvidenceAndOriginalBounds(t *testing.T) {
	a, b := NewPackage(), NewPackage()
	a.Name, b.Name = "a", "b"
	claims := []*Addr{
		{AddrPos: &AddrPos{Addr: 100, Size: 16, Type: AddrTypeData}, Pkg: a, SourceType: AddrSourceSymbol, Boundary: "symbol"},
		{AddrPos: &AddrPos{Addr: 100, Size: 8, Type: AddrTypeData}, Pkg: a, SourceType: AddrSourceDwarf, Boundary: "declared"},
		{AddrPos: &AddrPos{Addr: 100, Size: 24, Type: AddrTypeData}, Pkg: b, SourceType: AddrSourceSymbol, Boundary: "symbol"},
	}
	space := AddrSpace{}
	for _, claim := range claims {
		space.Insert(claim)
		space.Insert(claim)
	}
	got := summarizeEvidence(space.All)
	require.Equal(t, []evidenceSummary{
		{100, 8, "a", AddrSourceDwarf, AddrRoleDefinition, "declared", ""},
		{100, 16, "a", AddrSourceSymbol, AddrRoleDefinition, "symbol", ""},
		{100, 24, "b", AddrSourceSymbol, AddrRoleDefinition, "symbol", ""},
	}, got)
	preferred, ok := space.Get(100)
	require.True(t, ok)
	require.Equal(t, AddrSourceDwarf, preferred.SourceType, "typed definitions must remain visible to header validation")
	require.Equal(t, uint64(8), preferred.Size)
	coverage, err := MergeAndCleanCoverage([]AddrCoverage{space.ToDirtyCoverage()})
	require.NoError(t, err)
	require.Len(t, coverage, 1)
	require.Equal(t, uint64(24), coverage[0].Pos.Size)
	require.Equal(t, got, summarizeEvidence(space.All), "union construction must not widen evidence")
	require.Equal(t, uint64(8), claims[1].Size)
}

func TestAddrSpaceEvidenceIsIndependentOfInsertionOrder(t *testing.T) {
	a, b := NewPackage(), NewPackage()
	a.Name, b.Name = "a", "b"
	claims := []*Addr{
		{AddrPos: &AddrPos{Addr: 101, Size: 4, Type: AddrTypeData}, Pkg: a, SourceType: AddrSourceSymbol},
		{AddrPos: &AddrPos{Addr: 100, Size: 16, Type: AddrTypeData}, Pkg: a, SourceType: AddrSourceSymbol},
		{AddrPos: &AddrPos{Addr: 100, Size: 8, Type: AddrTypeData}, Pkg: b, SourceType: AddrSourceDwarf},
		{AddrPos: &AddrPos{Addr: 100, Size: 40, Type: AddrTypeData}, Pkg: b, SourceType: AddrSourceStaticRead, Role: AddrRoleReference, Boundary: "instruction"},
	}
	var expected []evidenceSummary
	for _, order := range [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {1, 3, 0, 2}, {2, 0, 3, 1}} {
		space := AddrSpace{}
		for _, i := range order {
			space.Insert(claims[i])
		}
		if expected == nil {
			expected = summarizeEvidence(space.All)
		}
		require.Equal(t, expected, summarizeEvidence(space.All))
		preferred, _ := space.Get(100)
		require.Equal(t, AddrSourceDwarf, preferred.SourceType)
		require.Equal(t, uint64(8), preferred.Size)
	}
}

func TestAddrSpaceMergeCopiesFlatEvidence(t *testing.T) {
	first := AddrSpace{}
	original := &Addr{AddrPos: &AddrPos{Addr: 100, Size: 8, Type: AddrTypeData}, SourceType: AddrSourceSymbol}
	first.Insert(original)
	first.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: 16, Type: AddrTypeData}, SourceType: AddrSourceDwarf})
	expected := summarizeEvidence(first.All)
	merged := MergeAddrSpace(first, first)
	require.Equal(t, expected, summarizeEvidence(merged.All))
	for addr := range merged.All {
		require.Empty(t, addr.evidence, "stored claims must not contain nested representative state")
		addr.Size = 99
	}
	original.Size = 88
	require.Equal(t, expected, summarizeEvidence(first.All))
	first.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: 4, Type: AddrTypeData}, SourceType: AddrSourceTyp})
	require.Len(t, slices.Collect(first.All), 3)
	require.Len(t, slices.Collect(merged.All), 2)
	visits := 0
	first.All(func(*Addr) bool { visits++; return false })
	require.Equal(t, 1, visits)
}

func TestStaticAccessPreservesExistingDefinitionAndOtherPackageReference(t *testing.T) {
	known := NewKnownAddr(testEvidenceStore())
	definition, consumer := NewPackage(), NewPackage()
	definition.Name, consumer.Name = "owner", "consumer"
	symbol := NewSymbol("owner.table", 100, 32, AddrTypeData)
	definition.AddSymbol(symbol, known.InsertSymbol(symbol, definition))
	known.BuildSymbolCoverage()
	fn := &Function{Name: "consumer.read", Addr: 2000, CodeSize: 8}
	fn.Init()
	consumer.AddFuncIfNotExists("consumer.go", fn)
	for range 2 {
		known.InsertDisasm(108, 8, fn, AddrSourceStaticRead)
		known.InsertDisasm(108, 8, fn, AddrSourceStaticCopy)
		known.InsertDisasm(108, 8, fn, AddrSourceDisasm)
	}
	refs := summarizeEvidence(fn.disasm.All)
	require.Equal(t, []evidenceSummary{
		{108, 8, "consumer", AddrSourceStaticCopy, AddrRoleReference, "instruction", "consumer.read"},
		{108, 8, "consumer", AddrSourceStaticRead, AddrRoleReference, "instruction", "consumer.read"},
	}, refs, "exact reads/copies are retained while overlapping heuristic strings are rejected")
	var dataClaims []evidenceSummary
	for _, claim := range summarizeEvidence(consumer.OwnAddresses) {
		if claim.addr == 108 {
			dataClaims = append(dataClaims, claim)
		}
	}
	require.Equal(t, refs, dataClaims)
	require.Equal(t, []evidenceSummary{{100, 32, "owner", AddrSourceSymbol, AddrRoleDefinition, "symbol", "owner.table"}}, summarizeEvidence(definition.OwnAddresses))
	require.NoError(t, definition.AssignPackageSize())
	require.NoError(t, consumer.AssignPackageSize())
	require.Equal(t, uint64(32), definition.Size)
	require.Equal(t, uint64(16), consumer.Size, "8 code bytes and one 8-byte data range")
}

func TestPackageOwnAddressesPreservesTextAndSymbolEvidence(t *testing.T) {
	known := NewKnownAddr(testEvidenceStore())
	p := NewPackage()
	p.Name = "main"
	fn := &Function{Name: "main.f", Addr: 2000, CodeSize: 16, Source: AddrSourceGoPclntab}
	fn.Init()
	p.AddFuncIfNotExists("main.go", fn)
	known.InsertTextFromPclnTab(2000, 16, fn)
	known.InsertTextFromDWARF(2000, 12, fn)
	known.InsertTextFromDWARF(2000, 12, fn)
	for _, source := range []AddrSourceType{AddrSourceSymbol, AddrSourceDwarf, AddrSourceSymbol} {
		symbol := NewSymbol("main.data", 100, 8, AddrTypeData)
		p.AddSymbol(symbol, known.InsertSymbol(symbol, p, source))
	}
	claims := summarizeEvidence(p.OwnAddresses)
	require.Len(t, claims, 4)
	require.Contains(t, claims, evidenceSummary{100, 8, "main", AddrSourceSymbol, AddrRoleDefinition, "symbol", "main.data"})
	require.Contains(t, claims, evidenceSummary{100, 8, "main", AddrSourceDwarf, AddrRoleDefinition, "declared", "main.data"})
	require.Contains(t, claims, evidenceSummary{2000, 12, "main", AddrSourceDwarf, AddrRoleDefinition, "declared", "main.f"})
	require.Contains(t, claims, evidenceSummary{2000, 16, "main", AddrSourceGoPclntab, AddrRoleDefinition, "encoded", "main.f"})
	require.NoError(t, p.AssignPackageSize())
	require.Equal(t, uint64(24), p.Size)
}

func TestSymbolEvidenceOverridesAreAppliedBeforeSnapshot(t *testing.T) {
	known := NewKnownAddr(testEvidenceStore())
	p := NewPackage()
	p.Name = "main"
	symbol := NewSymbol("main.pointer.data", 100, 8, AddrTypeData)
	symbol.Role, symbol.Boundary = AddrRoleReference, "estimated"
	p.AddSymbol(symbol, known.InsertSymbolFromDWARF(symbol, p))
	preferred, ok := known.SymbolAddrSpace.Get(100)
	require.True(t, ok)
	require.Equal(t, AddrRoleReference, preferred.Role)
	require.Equal(t, "estimated", preferred.Boundary)
	require.Equal(t, []evidenceSummary{{100, 8, "main", AddrSourceDwarf, AddrRoleReference, "estimated", "main.pointer.data"}}, summarizeEvidence(p.OwnAddresses))
	headerData := NewSymbol("main.string.data", 200, 12, AddrTypeData)
	addr := known.InsertSymbol(headerData, p, AddrSourceStaticHeader)
	require.Equal(t, AddrRoleReference, addr.Role)
	require.Equal(t, "heuristic", addr.Boundary)
}

func TestSharedTypePathEvidenceDoesNotGrowWithAliases(t *testing.T) {
	p := NewPackage()
	p.Name = "example/pkg"
	space := AddrSpace{}
	for i := 999; i >= 0; i-- {
		claim := &Addr{
			AddrPos: &AddrPos{Addr: 100, Size: 32, Type: AddrTypeData},
			Pkg:     p, SourceType: AddrSourceTyp, Boundary: "encoded",
			Symbol: NewSymbol(fmt.Sprintf("type.pkgpath:T%04d", i), 100, 32, AddrTypeData),
		}
		space.Insert(claim)
	}
	claims := slices.Collect(space.All)
	require.Len(t, claims, 1)
	require.Equal(t, "type.pkgpath:T0000", claims[0].Symbol.Name)
	for i := range 1000 {
		claim := &Addr{
			AddrPos: &AddrPos{Addr: 100, Size: 32, Type: AddrTypeData},
			Pkg:     p, SourceType: AddrSourceTyp, Boundary: "encoded",
			Symbol: NewSymbol(fmt.Sprintf("type.pkgpath:T%04d", i), 100, 32, AddrTypeData),
		}
		space.Insert(claim)
	}
	require.Equal(t, summarizeEvidence(space.All), summarizeEvidence(MergeAddrSpace(space, space).All))
	require.Len(t, slices.Collect(space.All), 1)
}

func BenchmarkAddrSpaceSharedTypePathEvidence(b *testing.B) {
	p := NewPackage()
	p.Name = "example/pkg"
	claims := make([]*Addr, 1000)
	for i := range claims {
		claims[i] = &Addr{
			AddrPos: &AddrPos{Addr: 100, Size: 32, Type: AddrTypeData},
			Pkg:     p, SourceType: AddrSourceTyp, Boundary: "encoded",
			Symbol: NewSymbol(fmt.Sprintf("type.pkgpath:T%04d", i), 100, 32, AddrTypeData),
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		space := AddrSpace{}
		for _, claim := range claims {
			space.Insert(claim)
		}
		if len(slices.Collect(space.All)) != 1 {
			b.Fatal("shared path created duplicate evidence")
		}
	}
}
