package dwarf

import (
	"debug/dwarf"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExternalDefinitionIsNotADeclaration(t *testing.T) {
	e := &dwarf.Entry{Tag: dwarf.TagSubprogram, Field: []dwarf.Field{
		{Attr: dwarf.AttrExternal, Val: true}, {Attr: dwarf.AttrLowpc, Val: uint64(0x1000)}, {Attr: dwarf.AttrHighpc, Val: int64(32)},
	}}
	require.False(t, EntryShouldIgnore(e))
	e.Field = append(e.Field, dwarf.Field{Attr: dwarf.AttrDeclaration, Val: true})
	require.True(t, EntryShouldIgnore(e))
}
