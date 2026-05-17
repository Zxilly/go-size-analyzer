package knowninfo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func newSymbolTestKnownInfo() *KnownInfo {
	sects := entity.NewStore()
	sects.Sections[".data"] = &entity.Section{
		Name:        ".data",
		Size:        0x100,
		FileSize:    0x100,
		Addr:        0x1000,
		AddrEnd:     0x1100,
		ContentType: entity.SectionContentData,
	}
	sects.BuildCache()

	k := &KnownInfo{
		Sects:       sects,
		VersionFlag: VersionFlag{Meq120: true},
	}
	k.Deps = NewDependencies(k)
	k.KnownAddr = entity.NewKnownAddr(sects)
	return k
}

func TestMarkSymbolRoutesGeneratedSymbols(t *testing.T) {
	k := newSymbolTestKnownInfo()

	k.MarkSymbol("go:itab.*bytes.Buffer,io.Reader", 0x1000, 4, entity.AddrTypeData)
	k.MarkSymbol("go:buildinfo", 0x1004, 4, entity.AddrTypeData)
	k.MarkSymbol("type:.namedata.foo.", 0x1008, 4, entity.AddrTypeData)

	itabs, ok := k.Deps.GetPackage("runtime/itabs")
	require.True(t, ok)
	require.Len(t, itabs.Symbols, 1)

	generated, ok := k.Deps.GetPackage("runtime/generated")
	require.True(t, ok)
	require.Len(t, generated.Symbols, 1)

	types, ok := k.Deps.GetPackage("runtime/types")
	require.True(t, ok)
	require.Len(t, types.Symbols, 1)
}

func TestFallbackPackageNameIncludesType(t *testing.T) {
	require.Equal(t, "<unnamed:generated>", fallbackPackageName(entity.PackageTypeGenerated))
}
