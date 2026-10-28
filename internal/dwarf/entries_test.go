package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/blacktop/go-macho"
	"github.com/stretchr/testify/require"
)

func TestReadAnalysisEntriesMatchesMachOLegacyTargets(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "bins", "bin-darwin-1.27-amd64-pie-cgo")
	file, err := macho.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("external-link Mach-O fixture is not installed")
	}
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	d, err := wrapper.NewMachoWrapper(file).DWARF()
	require.NoError(t, err)
	wantKinds, err := ReadGoTypeKinds(d)
	require.NoError(t, err)
	wantEntries := legacyAnalysisTargets(t, d)
	got, err := ReadAnalysisEntries(d)
	require.NoError(t, err)
	require.True(t, got.CanReplay)
	require.Equal(t, wantKinds, got.Kinds)
	require.Equal(t, wantEntries, replayAnalysisTargets(got))
}

func TestReadAnalysisEntriesRetainsNestedKindsAndTargetOrder(t *testing.T) {
	unit := []byte{1, 'p', 0, 0x16, 0, 2, 'f', 0, 3, 'l', 0}
	// A named Go string is nested in a skipped function. Its kind must still
	// be available before any package variables are attributed.
	unit = append(unit, 5)
	unit = append(unit, "NamedString\x00"...)
	unit = append(unit, 16, 24, 4, 3, 'n', 0, 0, 0, 3, 'g', 0, 4, 3, 's', 0, 0, 6, 'h', 0, 0)
	invalidUnit := []byte{7, 'q', 0, 3, 'i', 0, 0}
	d := analysisEntriesFixture(t, unit, invalidUnit)
	wantKinds, err := ReadGoTypeKinds(d)
	require.NoError(t, err)
	got, err := ReadAnalysisEntries(d)
	require.NoError(t, err)
	require.True(t, got.CanReplay)
	require.Len(t, got.Kinds, 1)
	require.Equal(t, wantKinds, got.Kinds)
	require.Equal(t, legacyAnalysisTargets(t, d), replayAnalysisTargets(got))
	var names []string
	for _, entry := range replayAnalysisTargets(got) {
		name, ok := entry.Val(dwarf.AttrName).(string)
		require.True(t, ok)
		names = append(names, name)
	}
	require.Equal(t, []string{"p", "f", "g", "s", "h", "q"}, names)
}

func TestReadAnalysisEntriesDoesNotHideErrorsInSkippedChildren(t *testing.T) {
	for _, body := range [][]byte{
		{1, 'p', 0, 0x16, 0, 2, 'f', 0, 255, 1}, // unknown abbreviation in function body
		{1, 'p', 0, 0x16, 0, 2, 'f', 0, 3, 'x'}, // unterminated local variable name
	} {
		d := analysisEntriesFixture(t, body)
		_, wantErr := ReadGoTypeKinds(d)
		require.Error(t, wantErr)
		got, err := ReadAnalysisEntries(d)
		require.Nil(t, got)
		require.EqualError(t, err, wantErr.Error())
	}
}

func TestReadAnalysisEntriesFallsBackForUnusualReaderStructure(t *testing.T) {
	for name, body := range map[string][]byte{
		"sibling on skipped descendant": {1, 'p', 0, 0x16, 0, 2, 'f', 0, 8, 0, 0, 0, 0, 0, 0, 0},
		"unclosed compile unit":         {1, 'p', 0, 0x16, 0, 3, 'g', 0},
		"extra closing entry":           {1, 'p', 0, 0x16, 0, 0, 0},
		"nested compile unit":           {1, 'p', 0, 0x16, 0, 1, 'q', 0, 0x16, 0, 0, 0},
		"partial unit":                  {9, 0},
	} {
		t.Run(name, func(t *testing.T) {
			d := analysisEntriesFixture(t, body)
			wantKinds, err := ReadGoTypeKinds(d)
			require.NoError(t, err)
			got, err := ReadAnalysisEntries(d)
			require.NoError(t, err)
			require.False(t, got.CanReplay)
			require.Nil(t, got.Units)
			require.Equal(t, wantKinds, got.Kinds)
		})
	}
}

func validAnalysisUnit(entry *dwarf.Entry) bool {
	_, language := entry.Val(dwarf.AttrLanguage).(int64)
	_, name := entry.Val(dwarf.AttrName).(string)
	return language && name
}

func legacyAnalysisTargets(t *testing.T, d *dwarf.Data) []*dwarf.Entry {
	t.Helper()
	var targets []*dwarf.Entry
	r := d.Reader()
	valid := false
	for {
		entry, err := r.Next()
		require.NoError(t, err)
		if entry == nil {
			return targets
		}
		switch entry.Tag {
		case dwarf.TagCompileUnit:
			targets = append(targets, entry)
			valid = validAnalysisUnit(entry)
			if !valid {
				r.SkipChildren()
			}
		case dwarf.TagSubprogram, dwarf.TagVariable:
			if valid && !EntryShouldIgnore(entry) {
				targets = append(targets, entry)
			}
			if entry.Tag == dwarf.TagSubprogram {
				r.SkipChildren()
			}
		default:
		}
	}
}

func replayAnalysisTargets(entries *AnalysisEntries) []*dwarf.Entry {
	var targets []*dwarf.Entry
	for _, unit := range entries.Units {
		targets = append(targets, unit.Unit)
		if validAnalysisUnit(unit.Unit) {
			for _, entry := range unit.Entries {
				if !EntryShouldIgnore(entry) {
					targets = append(targets, entry)
				}
			}
		}
	}
	return targets
}

func analysisEntriesFixture(t *testing.T, units ...[]byte) *dwarf.Data {
	t.Helper()
	abbrev := []byte{
		1, 0x11, 1, 0x03, 0x08, 0x13, 0x05, 0, 0, // CU: name/string, language/data2
		2, 0x2e, 1, 0x03, 0x08, 0, 0, // function with children
		3, 0x34, 0, 0x03, 0x08, 0, 0, // variable
		4, 0x0b, 1, 0, 0, // lexical block
		5, 0x13, 0, 0x03, 0x08, 0x0b, 0x0b, 0x80, 0x52, 0x0b, 0, 0, // Go struct: name, size, kind
		6, 0x2e, 0, 0x03, 0x08, 0, 0, // function without children
		7, 0x11, 1, 0x03, 0x08, 0, 0, // CU without language
		8, 0x0b, 1, 0x01, 0x13, 0, 0, // block with sibling/ref4
		9, 0x3c, 1, 0, 0, // partial unit
		0,
	}
	var info []byte
	for _, entries := range units {
		body := append([]byte{4, 0, 0, 0, 0, 0, 8}, entries...)
		info = binary.LittleEndian.AppendUint32(info, uint32(len(body)))
		info = append(info, body...)
	}
	d, err := dwarf.New(abbrev, nil, nil, info, nil, nil, nil, nil)
	require.NoError(t, err)
	return d
}
