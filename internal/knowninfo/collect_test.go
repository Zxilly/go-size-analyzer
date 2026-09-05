//go:build !js && !wasm

package knowninfo_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/test"
	"github.com/stretchr/testify/require"
)

func TestUnattributedMetadataRemainsVisible(t *testing.T) {
	r := test.GetTestResult(t)
	debugFound, symbolsFound := false, false
	for _, s := range r.Sections {
		if s.Debug {
			debugFound = true
			require.Positive(t, s.FileSize)
			require.Zero(t, s.KnownSize, "debug bytes have not been attributed to any package")
		}
		if s.Name == ".symtab" {
			symbolsFound = true
			require.Positive(t, s.FileSize)
			require.Zero(t, s.KnownSize, "symbol table must retain its own area")
		}
	}
	require.True(t, debugFound)
	require.True(t, symbolsFound)
}
