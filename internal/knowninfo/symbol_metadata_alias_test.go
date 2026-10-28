package knowninfo

import (
	"slices"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/stretchr/testify/require"
)

func TestAttributeSymbolMetadataLeavesAmbiguousAliasesRecognized(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		name := "forward"
		if reversed {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			k := newMetadataAliasInfo()
			aliases := []string{"a.a", "a.z"}
			if reversed {
				slices.Reverse(aliases)
			}
			addMetadataAliasDefinitions(t, k, "a", aliases...)
			addMetadataAliasDefinitions(t, k, "b", "b.b")

			ledger := entity.NewFileLedger(30)
			require.NoError(t, ledger.Add(entity.FileClaim{FileRange: entity.FileRange{Size: 30}, Class: 1, Source: "symbol-table"}))
			var records []wrapper.SymbolMetadata
			for i, name := range []string{"a.a", "a.z", "b.b"} {
				records = append(records, wrapper.SymbolMetadata{
					Name: name, Addr: 0x1000, Type: entity.AddrTypeData,
					Ranges: []wrapper.MetadataRange{{FileRange: entity.FileRange{Offset: uint64(i * 10), Size: 10}, Source: "symbol_table"}},
				})
			}
			require.NoError(t, k.attributeSymbolMetadata(ledger, records))
			coverage := ledger.Finish(true)
			require.Equal(t, uint64(20), coverage.Attributed)
			require.Equal(t, uint64(10), coverage.Recognized)
			require.Zero(t, coverage.Shared)
			require.Equal(t, map[string]uint64{"a": 10, "b": 10}, coverage.PackageSizes)
			require.Len(t, coverage.Regions, 3)
			require.Equal(t, []string{"a"}, coverage.Regions[0].Owners)
			require.Equal(t, "recognized", coverage.Regions[1].Class, "the omitted a.z alias must not acquire both same-address packages")
			require.Empty(t, coverage.Regions[1].Owners)
			require.Equal(t, []string{"b"}, coverage.Regions[2].Owners)
		})
	}
}

func TestAttributeSymbolMetadataAllowsSinglePackageAlias(t *testing.T) {
	k := newMetadataAliasInfo()
	addMetadataAliasDefinitions(t, k, "a", "a.a")
	ledger := entity.NewFileLedger(10)
	require.NoError(t, k.attributeSymbolMetadata(ledger, []wrapper.SymbolMetadata{{
		Name: "a.z", Addr: 0x1000, Type: entity.AddrTypeData,
		Ranges: []wrapper.MetadataRange{{FileRange: entity.FileRange{Size: 10}, Source: "symbol_table"}},
	}}))
	coverage := ledger.Finish(true)
	require.Equal(t, uint64(10), coverage.Attributed)
	require.Equal(t, map[string]uint64{"a": 10}, coverage.PackageSizes)
	require.Equal(t, []string{"a"}, coverage.Regions[0].Owners)
}

func newMetadataAliasInfo() *KnownInfo {
	store := entity.NewStore()
	store.Sections["data"] = &entity.Section{Addr: 0x1000, Size: 16, FileSize: 16, ContentType: entity.SectionContentData}
	store.BuildCache()
	k := &KnownInfo{Sects: store, KnownAddr: entity.NewKnownAddr(store)}
	k.Deps = NewDependencies(k)
	return k
}

func addMetadataAliasDefinitions(t *testing.T, k *KnownInfo, packageName string, names ...string) {
	t.Helper()
	p := entity.NewPackage()
	p.Name = packageName
	k.Deps.Trie.Put(packageName, p)
	for _, name := range names {
		symbol := entity.NewSymbol(name, 0x1000, 16, entity.AddrTypeData)
		addr := k.KnownAddr.InsertSymbol(symbol, p)
		require.NotNil(t, addr)
		p.AddSymbol(symbol, addr)
	}
}
