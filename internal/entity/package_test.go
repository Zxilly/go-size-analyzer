package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestPackageMergePreservesDistinctMethodAddresses(t *testing.T) {
	a, b := entity.NewPackage(), entity.NewPackage()
	a.Name = "example"
	b.Name = "example"
	require.True(t, a.AddFuncIfNotExists("types.go", &entity.Function{Name: "Read", Receiver: "A", Type: entity.FuncTypeMethod, Addr: 100, CodeSize: 10}))
	require.True(t, b.AddFuncIfNotExists("types.go", &entity.Function{Name: "Read", Receiver: "B", Type: entity.FuncTypeMethod, Addr: 200, CodeSize: 20}))
	a.Merge(b)
	require.Equal(t, 2, a.FuncCount())
	require.Equal(t, uint64(30), a.GetFunctionSizeRecursive())
	// DWARF and pclntab spell the same function differently; its address identifies it.
	require.False(t, a.AddFuncIfNotExists("other.go", &entity.Function{Name: "example.A.Read", Addr: 100, CodeSize: 10}))
	a.Merge(b)
	require.Len(t, a.Files, 1)
	require.Len(t, a.Files[0].Functions, 2)
	require.Equal(t, uint64(30), a.GetFunctionSizeRecursive())
}

func TestPackageKeepsSameNamedFunctionsInDifferentFiles(t *testing.T) {
	p := entity.NewPackage()
	require.True(t, p.AddFuncIfNotExists("a.c", &entity.Function{Name: "helper", Addr: 100, CodeSize: 10}))
	require.True(t, p.AddFuncIfNotExists("b.c", &entity.Function{Name: "helper", Addr: 200, CodeSize: 20}))
	require.Equal(t, 2, p.FuncCount())
}
