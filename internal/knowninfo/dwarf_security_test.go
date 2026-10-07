package knowninfo

import (
	"debug/dwarf"
	"encoding/binary"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func reversedRangeData(t *testing.T) (*dwarf.Data, *dwarf.Entry) {
	t.Helper()
	// DWARF 4 compile unit with a subprogram whose absolute high_pc precedes low_pc.
	abbrev := []byte{1, 0x11, 1, 0, 0, 2, 0x2e, 0, 3, 8, 0x11, 1, 0x12, 1, 0, 0, 0}
	body := []byte{4, 0, 0, 0, 0, 0, 8, 1, 2, 'b', 'a', 'd', 0}
	body = binary.LittleEndian.AppendUint64(body, 100)
	body = binary.LittleEndian.AppendUint64(body, 90)
	body = append(body, 0)
	info := binary.LittleEndian.AppendUint32(nil, uint32(len(body)))
	info = append(info, body...)
	d, err := dwarf.New(abbrev, nil, nil, info, nil, nil, nil, nil)
	require.NoError(t, err)
	r := d.Reader()
	_, err = r.Next()
	require.NoError(t, err)
	entry, err := r.Next()
	require.NoError(t, err)
	return d, entry
}

func TestReversedDwarfRangeRejectedBeforeFunctionRegistration(t *testing.T) {
	d, entry := reversedRangeData(t)
	ranges, err := d.Ranges(entry)
	require.NoError(t, err)
	require.Equal(t, [][2]uint64{{100, 90}}, ranges)
	k := &KnownInfo{KnownAddr: entity.NewKnownAddr(entity.NewStore())}
	pkg := entity.NewPackage()
	err = k.AddDwarfSubProgram(true, d, entry, pkg, func(*dwarf.Entry) string { return "bad.go" })
	require.ErrorContains(t, err, "reversed range")
	require.Zero(t, pkg.FuncCount(), "reversed ranges must never become wrapped function sizes")
}

func TestDwarfRangeSizeOverflowRejected(t *testing.T) {
	// A real DWARF 4 range list whose individually ordered intervals overflow the sum.
	abbrev := []byte{1, 0x11, 1, 0, 0, 2, 0x2e, 0, 3, 8, 0x55, 0x17, 0, 0, 0}
	body := []byte{4, 0, 0, 0, 0, 0, 8, 1, 2, 'b', 'a', 'd', 0, 0, 0, 0, 0, 0}
	info := binary.LittleEndian.AppendUint32(nil, uint32(len(body)))
	info = append(info, body...)
	var encoded []byte
	expected := [][2]uint64{{1, ^uint64(0)}, {1, 3}}
	for _, r := range expected {
		encoded = binary.LittleEndian.AppendUint64(encoded, r[0])
		encoded = binary.LittleEndian.AppendUint64(encoded, r[1])
	}
	encoded = append(encoded, make([]byte, 16)...)
	d, err := dwarf.New(abbrev, nil, nil, info, nil, nil, encoded, nil)
	require.NoError(t, err)
	reader := d.Reader()
	_, err = reader.Next()
	require.NoError(t, err)
	entry, err := reader.Next()
	require.NoError(t, err)
	ranges, err := d.Ranges(entry)
	require.NoError(t, err)
	require.Equal(t, expected, ranges)
	k := &KnownInfo{KnownAddr: entity.NewKnownAddr(entity.NewStore())}
	pkg := entity.NewPackage()
	err = k.AddDwarfSubProgram(true, d, entry, pkg, func(*dwarf.Entry) string { return "bad.go" })
	require.ErrorContains(t, err, "range size overflow")
	require.Zero(t, pkg.FuncCount())
}
