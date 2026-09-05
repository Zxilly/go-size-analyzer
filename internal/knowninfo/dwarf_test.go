package knowninfo

import (
	"debug/dwarf"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeGetEntryValReturnsValueOnSuccess(t *testing.T) {
	entry := &dwarf.Entry{}

	value, ok := safeGetEntryVal[int](entry, dwarf.Attr(1), "test attribute", false)
	assert.False(t, ok)
	assert.Zero(t, value)
}

func TestDwarfOnlyPackageIsRegistered(t *testing.T) {
	k := &KnownInfo{}
	k.Deps = NewDependencies(k)
	cu := &dwarf.Entry{Tag: dwarf.TagCompileUnit, Field: []dwarf.Field{
		{Attr: dwarf.AttrLanguage, Val: int64(0x16)}, {Attr: dwarf.AttrName, Val: "example.com/dataonly"},
	}}
	pkg := k.GetPackageFromDwarfCompileUnit(cu)
	stored, ok := k.Deps.GetPackage("example.com/dataonly")
	require.True(t, ok)
	require.Same(t, pkg, stored)
}
