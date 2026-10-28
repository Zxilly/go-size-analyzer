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
			require.Positive(t, s.KnownSize, "records for established package definitions are attributed")
			require.Less(t, s.KnownSize, s.FileSize, "unattributed symbol records must retain a visible section area")
		}
	}
	require.True(t, debugFound)
	require.True(t, symbolsFound)
}
