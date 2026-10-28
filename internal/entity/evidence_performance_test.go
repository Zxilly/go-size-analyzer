package entity

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func benchmarkAddressPackage() *Package {
	p := NewPackage()
	p.Name = "example/pkg"
	for i := range 1000 {
		entry := uint64(0x100000 + i*128)
		fn := &Function{Name: fmt.Sprintf("example/pkg.f%d", i), Addr: entry, CodeSize: 64, Source: AddrSourceGoPclntab}
		fn.Init()
		p.AddFuncIfNotExists("package.go", fn)
		for j := range 8 {
			fn.PclnRanges = append(fn.PclnRanges, AddrPos{Addr: uint64(0x200000 + i*128 + j*16), Size: 12, Type: AddrTypeData})
		}
		fn.codeEvidence.Insert(&Addr{AddrPos: &AddrPos{Addr: entry, Size: 64, Type: AddrTypeText}, Pkg: p, Function: fn, SourceType: AddrSourceGoPclntab})
		fn.codeEvidence.Insert(&Addr{AddrPos: &AddrPos{Addr: entry, Size: 60, Type: AddrTypeText}, Pkg: p, Function: fn, SourceType: AddrSourceSymbol})
		fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: uint64(0x300000 + i*16), Size: 16, Type: AddrTypeData}, Pkg: p, Function: fn, SourceType: AddrSourceStaticRead, Role: AddrRoleReference})
	}
	return p
}

func BenchmarkPackageOwnAddresses(b *testing.B) {
	p := benchmarkAddressPackage()
	b.ReportAllocs()
	for b.Loop() {
		var count int
		var size uint64
		for addr := range p.OwnAddresses {
			count++
			size += addr.Size
		}
		if count != 11000 || size != 236000 {
			b.Fatalf("unexpected evidence: %d claims, %d bytes", count, size)
		}
	}
}

func BenchmarkPackageOwnAddressValues(b *testing.B) {
	p := benchmarkAddressPackage()
	b.ReportAllocs()
	for b.Loop() {
		var count int
		var size uint64
		for addr := range p.OwnAddressValues {
			count++
			size += addr.Size
		}
		if count != 11000 || size != 236000 {
			b.Fatalf("unexpected evidence: %d claims, %d bytes", count, size)
		}
	}
}

func BenchmarkAddrSpaceInsertDistinctEvidence(b *testing.B) {
	p := NewPackage()
	p.Name = "example/pkg"
	var claims []*Addr
	for i := range 1000 {
		entry := uint64(100 + i*64)
		for j, source := range []AddrSourceType{AddrSourceSymbol, AddrSourceDwarf, AddrSourceTyp, AddrSourceStaticRead, AddrSourceStaticCopy, AddrSourceDisasm} {
			claim := &Addr{AddrPos: &AddrPos{Addr: entry, Size: uint64(8 + j*4), Type: AddrTypeData}, Pkg: p, SourceType: source, Symbol: NewSymbol("z", entry, 8, AddrTypeData)}
			claim.setSourceEvidence()
			claims = append(claims, claim)
		}
		alias := *claims[len(claims)-4]
		alias.Symbol = NewSymbol("a", entry, 8, AddrTypeData)
		claims = append(claims, &alias)
	}
	b.ReportAllocs()
	for b.Loop() {
		space := AddrSpace{}
		for _, claim := range claims {
			space.Insert(claim)
		}
		if len(space) != 1000 {
			b.Fatal("missing addresses")
		}
	}
}

func BenchmarkAddrSpaceAllSingleAddress(b *testing.B) {
	space := AddrSpace{}
	for _, size := range []uint64{8, 16, 24} {
		space.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: size, Type: AddrTypeData}, SourceType: AddrSourceSymbol})
	}
	b.ReportAllocs()
	for b.Loop() {
		var total uint64
		for addr := range space.All {
			total += addr.Size
		}
		if total != 48 {
			b.Fatal("missing evidence")
		}
	}
}

func TestPackageOwnAddressesKeepsRetainedSyntheticClaimsIndependent(t *testing.T) {
	p := NewPackage()
	p.Name = "p"
	fn := &Function{Name: "p.f", Addr: 100, CodeSize: 12, Source: AddrSourceDwarf, Ranges: []AddrPos{{Addr: 100, Size: 4}, {Addr: 200, Size: 8}}, PclnRanges: []AddrPos{{Addr: 300, Size: 3}, {Addr: 400, Size: 5}}}
	fn.Init()
	p.AddFuncIfNotExists("f.go", fn)
	first := slices.Collect(p.OwnAddresses)
	require.Len(t, first, 4)
	expected := []evidenceSummary{
		{100, 4, "p", AddrSourceDwarf, AddrRoleDefinition, "declared", "p.f"},
		{200, 8, "p", AddrSourceDwarf, AddrRoleDefinition, "declared", "p.f"},
		{300, 3, "p", AddrSourceGoPclntab, AddrRoleDefinition, "encoded", "p.f"},
		{400, 5, "p", AddrSourceGoPclntab, AddrRoleDefinition, "encoded", "p.f"},
	}
	require.Equal(t, expected, summarizeEvidence(func(yield func(*Addr) bool) {
		for _, addr := range first {
			if !yield(addr) {
				return
			}
		}
	}))
	second := slices.Collect(p.OwnAddresses)
	for i, addr := range first {
		require.NotSame(t, addr, second[i])
		require.NotSame(t, addr.AddrPos, second[i].AddrPos)
		addr.Addr, addr.Size, addr.Role = 900, 999, AddrRoleReference
	}
	require.Equal(t, expected, summarizeEvidence(p.OwnAddresses))
	require.Equal(t, uint64(3), fn.PclnRanges[0].Size)
	require.Equal(t, uint64(4), fn.Ranges[0].Size)
	visits := 0
	p.OwnAddresses(func(*Addr) bool { visits++; return false })
	require.Equal(t, 1, visits)
}

func TestAddrSpaceSingleAddressStopsDuringEvidence(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		space := AddrSpace{}
		for i := range count {
			space.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: uint64(i + 1), Type: AddrTypeData}})
		}
		var visits int
		space.All(func(*Addr) bool { visits++; return false })
		require.Equal(t, min(count, 1), visits)
	}
}

func TestAddrSpaceSingleAddressDoesNotVisitNewKeys(t *testing.T) {
	space := AddrSpace{}
	for _, size := range []uint64{8, 16} {
		space.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: size, Type: AddrTypeData}})
	}
	var visits int
	space.All(func(addr *Addr) bool {
		require.Equal(t, uint64(100), addr.Addr)
		visits++
		if visits == 1 {
			for i := range 50 {
				space.Insert(&Addr{AddrPos: &AddrPos{Addr: uint64(200 + i), Size: 1, Type: AddrTypeData}})
			}
		}
		return true
	})
	require.Equal(t, 2, visits)
}

func TestAddrSpaceMultipleAddressesVisitInOrderAndStop(t *testing.T) {
	space := AddrSpace{}
	for _, entry := range []uint64{300, 100, 200} {
		for _, size := range []uint64{16, 8} {
			space.Insert(&Addr{AddrPos: &AddrPos{Addr: entry, Size: size, Type: AddrTypeData}})
		}
	}
	expected := []AddrPos{
		{Addr: 100, Size: 8, Type: AddrTypeData},
		{Addr: 100, Size: 16, Type: AddrTypeData},
		{Addr: 200, Size: 8, Type: AddrTypeData},
		{Addr: 200, Size: 16, Type: AddrTypeData},
		{Addr: 300, Size: 8, Type: AddrTypeData},
		{Addr: 300, Size: 16, Type: AddrTypeData},
	}
	for limit := 1; limit <= len(expected); limit++ {
		var got []AddrPos
		space.All(func(addr *Addr) bool {
			got = append(got, *addr.AddrPos)
			return len(got) < limit
		})
		require.Equal(t, expected[:limit], got)
	}
}

func TestAddrSpaceAliasReplacementPreservesOtherClaimsAndPreferredLookup(t *testing.T) {
	p := NewPackage()
	p.Name = "p"
	claims := []*Addr{
		{AddrPos: &AddrPos{Addr: 100, Size: 12, Type: AddrTypeData}, Pkg: p, SourceType: AddrSourceSymbol, Boundary: "symbol", Symbol: NewSymbol("z", 100, 12, AddrTypeData)},
		{AddrPos: &AddrPos{Addr: 100, Size: 8, Type: AddrTypeData}, Pkg: p, SourceType: AddrSourceDwarf, Boundary: "declared", Symbol: NewSymbol("z", 100, 8, AddrTypeData)},
		{AddrPos: &AddrPos{Addr: 100, Size: 12, Type: AddrTypeData}, Pkg: p, SourceType: AddrSourceSymbol, Boundary: "symbol", Symbol: NewSymbol("a", 100, 12, AddrTypeData)},
		{AddrPos: &AddrPos{Addr: 100, Size: 8, Type: AddrTypeData}, Pkg: p, SourceType: AddrSourceDwarf, Boundary: "declared", Symbol: NewSymbol("a", 100, 8, AddrTypeData)},
	}
	for _, order := range [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {2, 1, 0, 3}, {1, 3, 0, 2}} {
		space := AddrSpace{}
		for _, i := range order {
			space.Insert(claims[i])
		}
		require.Equal(t, []evidenceSummary{
			{100, 8, "p", AddrSourceDwarf, AddrRoleDefinition, "declared", "a"},
			{100, 12, "p", AddrSourceSymbol, AddrRoleDefinition, "symbol", "a"},
		}, summarizeEvidence(space.All))
		preferred, ok := space.Get(100)
		require.True(t, ok)
		require.Equal(t, "a", preferred.Symbol.Name)
		require.Equal(t, AddrSourceDwarf, preferred.SourceType)
		claims[0].Size = 99
		require.Equal(t, uint64(12), slices.Collect(space.All)[1].Size)
		claims[0].Size = 12
	}
}

func TestPackageOwnAddressValuesMatchesPointerIteration(t *testing.T) {
	p := NewPackage()
	p.Name = "p"
	p.AddSymbolCoverage(&Addr{AddrPos: &AddrPos{Addr: 90, Size: 6, Type: AddrTypeData}, Pkg: p, Symbol: NewSymbol("p.data", 90, 6, AddrTypeData), SourceType: AddrSourceSymbol, Boundary: "symbol"})
	for _, entry := range []uint64{300, 100, 200} {
		fn := &Function{Name: fmt.Sprintf("p.f%d", entry), Addr: entry, CodeSize: 12, Source: AddrSourceDwarf, Ranges: []AddrPos{{Addr: entry, Size: 4}, {Addr: entry + 30, Size: 8}}, PclnRanges: []AddrPos{{Addr: entry + 1000, Size: 3}, {Addr: entry + 2000, Size: 5}}}
		fn.Init()
		p.AddFuncIfNotExists("f.go", fn)
		fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: entry + 3000, Size: 7, Type: AddrTypeData}, Pkg: p, Function: fn, SourceType: AddrSourceStaticRead, Role: AddrRoleReference, Boundary: "instruction"})
		if entry == 100 {
			for _, size := range []uint64{11, 12} {
				fn.codeEvidence.Insert(&Addr{AddrPos: &AddrPos{Addr: entry, Size: size, Type: AddrTypeText}, Pkg: p, Function: fn, SourceType: AddrSourceSymbol, Boundary: "symbol"})
			}
		}
		if entry == 300 {
			fn.Wasm = true
		}
	}
	p.ClearCache()
	var expected []AddrValue
	for addr := range p.OwnAddresses {
		expected = append(expected, AddrValue{AddrPos: *addr.AddrPos, Pkg: addr.Pkg, Function: addr.Function, Symbol: addr.Symbol, SourceType: addr.SourceType, Role: addr.Role, Boundary: addr.Boundary})
	}
	require.Len(t, expected, 14)
	values := slices.Collect(p.OwnAddressValues)
	require.Equal(t, expected, values)
	values[0].Addr, values[0].Size, values[0].SourceType, values[0].Role = 9000, 999, AddrSourceDisasm, AddrRoleReference
	require.Equal(t, expected, slices.Collect(p.OwnAddressValues), "mutating a value cannot change stored evidence")
	for limit := 1; limit <= len(expected); limit++ {
		var prefix []AddrValue
		p.OwnAddressValues(func(addr AddrValue) bool {
			prefix = append(prefix, addr)
			return len(prefix) < limit
		})
		require.Equal(t, expected[:limit], prefix)
	}
}
