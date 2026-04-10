//go:build !js && !wasm

package knowninfo_test

import (
	"testing"

	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/knowninfo"
	"github.com/Zxilly/go-size-analyzer/internal/test"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// buildKnownInfoWithVersion constructs a KnownInfo with a forced Go version,
// used to trigger moduledata selection errors in analyzer tests.
// go1.2 is a useful version: SetGoVersion accepts it, but minor=2 has no
// moduledata struct defined in selectModuleData, so Moduledata() returns a
// hard error (unlike ErrNoGoVersionFound, which is a soft-fail).
func buildKnownInfoWithVersion(t *testing.T, version string) *knowninfo.KnownInfo {
	t.Helper()
	path := test.GetTestBinPath(t)

	f, err := mmap.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	gf, err := gore.OpenReader(f)
	require.NoError(t, err)
	require.NoError(t, gf.SetGoVersion(version))

	k := &knowninfo.KnownInfo{
		Size:      uint64(f.Len()),
		BuildInfo: gf.BuildInfo,
		Gore:      gf,
		Wrapper:   wrapper.NewWrapper(gf.GetParsedFile()),
	}

	require.NoError(t, k.LoadSectionMap())
	k.KnownAddr = entity.NewKnownAddr(k.Sects)
	require.NoError(t, k.LoadGoreInfo(gf, false))

	return k
}
