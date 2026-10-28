package knowninfo

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/stretchr/testify/require"
)

func TestBuildFileCoverageProjectsPhysicalPackageAndSectionSizes(t *testing.T) {
	for _, details := range []bool{false, true} {
		name := "summary"
		if details {
			name = "details"
		}
		t.Run(name, func(t *testing.T) {
			k, data, sections, parent, child := fileAccountingFixture(t)
			coverage, err := k.BuildFileCoverage(bytes.NewReader(data), sections, details)
			require.NoError(t, err)
			require.Empty(t, coverage.Notes)
			require.Equal(t, uint64(103), coverage.Attributed, "24 data bytes, three 24-byte records, and one 7-byte name")
			require.Equal(t, uint64(15), coverage.Shared, "8 shared data bytes and one shared name")
			require.Equal(t, k.Size, coverage.Attributed+coverage.Recognized+coverage.Unclassified)
			require.Equal(t, uint64(103), parent.Size, "the parent includes the physical union of its own and child claims")
			require.Equal(t, uint64(47), child.Size)
			require.Equal(t, uint64(55), parent.MetadataSize, "the parent's two records share one encoded name")
			require.Equal(t, uint64(31), child.MetadataSize)
			require.Equal(t, uint64(24), sections[0].KnownSize)
			require.Equal(t, uint64(72), sections[1].KnownSize)
			require.Equal(t, uint64(7), sections[2].KnownSize)
			require.Zero(t, sections[3].KnownSize, "memory-only bytes have no physical projection")
			require.Zero(t, sections[4].KnownSize, "virtual sections have no physical projection")
			require.Equal(t, uint64(24), sections[1].FileSize-sections[1].KnownSize, "the null symbol record stays visible")
			require.Equal(t, uint64(72), coverage.BySource["symbol_table"])
			require.Equal(t, uint64(7), coverage.BySource["symbol_name"])
			if details {
				var sharedName *entity.FileRegion
				for i := range coverage.Regions {
					if coverage.Regions[i].Offset == 0x101 {
						sharedName = &coverage.Regions[i]
					}
				}
				require.NotNil(t, sharedName)
				require.Equal(t, uint64(7), sharedName.Size)
				require.Equal(t, []string{"p", "p/child"}, sharedName.Owners)
			} else {
				require.Empty(t, coverage.Regions)
			}
		})
	}
}

func TestBuildFileCoverageCanLeaveSymbolMetadataRecognized(t *testing.T) {
	k, data, sections, parent, child := fileAccountingFixture(t)
	k.SkipSymbolMetadata = true
	coverage, err := k.BuildFileCoverage(bytes.NewReader(data), sections, true)
	require.NoError(t, err)
	require.Empty(t, coverage.Notes)
	require.Equal(t, uint64(24), coverage.Attributed)
	require.Equal(t, uint64(24), parent.Size)
	require.Equal(t, uint64(16), child.Size)
	require.Zero(t, parent.MetadataSize)
	require.Zero(t, child.MetadataSize)
	require.Zero(t, sections[1].KnownSize)
	require.Zero(t, sections[2].KnownSize)
	require.NotContains(t, coverage.BySource, "symbol_table")
	require.NotContains(t, coverage.BySource, "symbol_name")
}

func fileAccountingFixture(t *testing.T) (k *KnownInfo, data []byte, sections []*entity.Section, parent, child *entity.Package) {
	t.Helper()
	data = make([]byte, 512)
	names := []byte("\x00Shared\x00unclaimed\x00")
	copy(data[0x100:], names)
	for i, addr := range []uint64{0x1000, 0x1000, 0x1008} {
		record := data[0x40+(i+1)*24:]
		binary.LittleEndian.PutUint32(record, 1)
		record[4] = byte(elf.STT_OBJECT)
		binary.LittleEndian.PutUint16(record[6:], 1)
		binary.LittleEndian.PutUint64(record[8:], addr)
	}
	sections = []*entity.Section{
		{Name: ".data", Addr: 0x1000, Offset: 0x180, Size: 32, FileSize: 32, ContentType: entity.SectionContentData},
		{Name: ".symtab", Offset: 0x40, Size: 96, FileSize: 96, ContentType: entity.SectionContentOther},
		{Name: ".strtab", Offset: 0x100, Size: uint64(len(names)), FileSize: uint64(len(names)), ContentType: entity.SectionContentOther},
		{Name: ".bss", Addr: 0x2000, Size: 64, KnownSize: 64, OnlyInMemory: true, ContentType: entity.SectionContentData},
		{Name: "virtual", Offset: 0x180, Size: 32, FileSize: 32, KnownSize: 32, VirtualSection: true},
	}
	store := entity.NewStore()
	for _, section := range sections {
		store.Sections[section.Name] = section
	}
	store.BuildCache()
	k = &KnownInfo{Size: uint64(len(data)), Sects: store, KnownAddr: entity.NewKnownAddr(store)}
	k.Wrapper = wrapper.NewWrapper(&elf.File{
		FileHeader: elf.FileHeader{Class: elf.ELFCLASS64, ByteOrder: binary.LittleEndian},
		Sections: []*elf.Section{
			{SectionHeader: elf.SectionHeader{Type: elf.SHT_NULL}},
			{SectionHeader: elf.SectionHeader{Name: ".data", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC | elf.SHF_WRITE, Addr: 0x1000, Offset: 0x180, Size: 32, FileSize: 32}},
			{SectionHeader: elf.SectionHeader{Name: ".symtab", Type: elf.SHT_SYMTAB, Offset: 0x40, Size: 96, FileSize: 96, Entsize: 24, Link: 3}},
			{SectionHeader: elf.SectionHeader{Name: ".strtab", Type: elf.SHT_STRTAB, Offset: 0x100, Size: uint64(len(names)), FileSize: uint64(len(names))}},
		},
	})
	k.Deps = NewDependencies(k)
	parent, child = entity.NewPackage(), entity.NewPackage()
	parent.Name, child.Name = "p", "p/child"
	parent.SubPackages["child"] = child
	k.Deps.TopPkgs[parent.Name] = parent
	for i, p := range []*entity.Package{parent, child} {
		k.Deps.Trie.Put(p.Name, p)
		p.Size, p.MetadataSize = 999, 999
		symbol := entity.NewSymbol("Shared", 0x1000+uint64(i)*8, 16, entity.AddrTypeData)
		addr := k.KnownAddr.InsertSymbol(symbol, p)
		require.NotNil(t, addr)
		p.AddSymbol(symbol, addr)
	}
	return k, data, sections, parent, child
}
