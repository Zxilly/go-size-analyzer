package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/stretchr/testify/require"
)

func pointerMemory(data []byte, order binary.ByteOrder) MemoryReader {
	valid := func(addr, size uint64) bool {
		return addr >= 0x1000 && addr-0x1000 <= uint64(len(data)) && size <= uint64(len(data))-(addr-0x1000)
	}
	return MemoryReader{ByteOrder: order, Limit: uint64(len(data)), Valid: valid, Read: func(addr, size uint64) ([]byte, error) {
		if !valid(addr, size) {
			return nil, wrapper.ErrAddrNotFound
		}
		return data[addr-0x1000 : addr-0x1000+size], nil
	}}
}

func TestTypedPointerStringWidthsAndByteOrder(t *testing.T) {
	for _, width := range []int64{4, 8} {
		for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			t.Run(fmt.Sprintf("%d/%s", width, order), func(t *testing.T) {
				data := make([]byte, 128)
				put := func(off int, n uint64) {
					if width == 4 {
						order.PutUint32(data[off:], uint32(n))
					} else {
						order.PutUint64(data[off:], n)
					}
				}
				put(0, 0x1020)
				put(32, 0x1060)
				put(32+int(width), 9)
				copy(data[96:], "pointee!!")
				target := stringLayout(width, width*2, width)
				ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: width}, Type: target}
				w := typedDataWalker{memory: pointerMemory(data, order), references: map[dwarf.Type]bool{}}
				w.walk(ptr, 0x1000, "root", 0)
				require.NoError(t, w.err)
				require.Equal(t, []Content{{Name: "root.*", Addr: 0x1020, Size: uint64(width * 2)}, {Name: "root.*.string", Addr: 0x1060, Size: 9}}, w.contents)
			})
		}
	}
}

func TestTypedPointerSharedCycleAndDepthBudget(t *testing.T) {
	data := make([]byte, 128)
	binary.LittleEndian.PutUint64(data, 0x1020)
	binary.LittleEndian.PutUint64(data[8:], 0x1020)
	binary.LittleEndian.PutUint64(data[32:], 0x1040)
	binary.LittleEndian.PutUint64(data[64:], 0x1020)
	node := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 8}, StructName: "Node"}
	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: node}
	node.Field = []*dwarf.StructField{{Name: "Next", Type: ptr}}
	root := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, Field: []*dwarf.StructField{{Name: "A", Type: ptr}, {Name: "B", Type: ptr, ByteOffset: 8}}}
	w := typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}}
	w.walk(root, 0x1000, "", 0)
	require.NoError(t, w.err)
	require.Len(t, w.contents, 2)
	require.Equal(t, uint64(0x1020), w.contents[0].Addr)
	require.Equal(t, uint64(0x1040), w.contents[1].Addr)
	require.Less(t, w.visited, 10)
	w = typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}}
	w.walk(ptr, 0x1000, "", 32)
	require.Empty(t, w.contents)
	w = typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}, visited: 8192}
	w.walk(ptr, 0x1000, "", 0)
	require.Empty(t, w.contents)
}

func TestTypedPointerRejectsInvalidOrRuntimeManagedTargets(t *testing.T) {
	data := make([]byte, 128)
	target := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, StructName: "Payload"}
	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: target}
	for name, address := range map[string]uint64{"nil": 0, "outside": 0x2000, "cross boundary": 0x1078, "overflow": ^uint64(0) - 7} {
		t.Run(name, func(t *testing.T) {
			binary.LittleEndian.PutUint64(data, address)
			w := typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}}
			w.walk(ptr, 0x1000, "", 0)
			require.NoError(t, w.err)
			require.Empty(t, w.contents)
		})
	}
	for _, typ := range []dwarf.Type{&dwarf.VoidType{}, &dwarf.FuncType{}, &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, StructName: "runtime.eface"}, &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, StructName: "runtime.iface"}, &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, Kind: "union"}, &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 16}, Incomplete: true}} {
		t.Run(fmt.Sprintf("%T/%s", typ, typ.String()), func(t *testing.T) {
			ptr.Type = typ
			binary.LittleEndian.PutUint64(data, 0x1020)
			w := typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}}
			w.walk(ptr, 0x1000, "", 0)
			require.Empty(t, w.contents)
		})
	}
	ptr.Type = target
	for _, kind := range []int64{18, 19, 20, 21, 26} {
		t.Run(fmt.Sprintf("Go kind %d", kind), func(t *testing.T) {
			memory := pointerMemory(data, binary.LittleEndian)
			memory.Kinds = map[dwarf.Type]int64{target: kind}
			w := typedDataWalker{memory: memory, references: map[dwarf.Type]bool{}}
			w.walk(ptr, 0x1000, "", 0)
			require.Empty(t, w.contents)
		})
	}
}

func TestTypedPointerPreservesOtherFieldsWhenTargetMalformed(t *testing.T) {
	data := make([]byte, 128)
	binary.LittleEndian.PutUint64(data, 0x1060)
	binary.LittleEndian.PutUint64(data[8:], 4)
	binary.LittleEndian.PutUint64(data[16:], 0x1040)
	malformed := stringLayout(3, 6, 3)
	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: malformed}
	root := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 24}, Field: []*dwarf.StructField{{Name: "Good", Type: stringLayout(8, 16, 8)}, {Name: "Bad", Type: ptr, ByteOffset: 16}}}
	w := typedDataWalker{memory: pointerMemory(data, binary.LittleEndian), references: map[dwarf.Type]bool{}}
	w.walk(root, 0x1000, "", 0)
	require.NoError(t, w.err)
	require.Equal(t, []Content{{Name: "Good.string", Addr: 0x1060, Size: 4}}, w.contents)
}

func TestTypedPointerAddressResolutionAndSharedScalar(t *testing.T) {
	data := make([]byte, 128)
	binary.LittleEndian.PutUint64(data, 0x20)
	binary.LittleEndian.PutUint64(data[8:], 0x20)
	byteType := &dwarf.UintType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "uint8", ByteSize: 1}}}
	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: byteType}
	root := &dwarf.ArrayType{Type: ptr, Count: 2, CommonType: dwarf.CommonType{ByteSize: 16}}
	memory := pointerMemory(data, binary.LittleEndian)
	memory.Resolve = func(addr uint64) uint64 { return addr + 0x1000 }
	w := typedDataWalker{memory: memory, references: map[dwarf.Type]bool{}}
	w.walk(root, 0x1000, "", 0)
	require.NoError(t, w.err)
	require.Equal(t, []Content{{Name: "[0].*", Addr: 0x20, Size: 1}}, w.contents)
}

func TestTypedPointerRejectsCyclicAndOverflowingArrayTypes(t *testing.T) {
	cyclic := &dwarf.ArrayType{Count: 1}
	cyclic.Type = cyclic
	byteType := &dwarf.UintType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "uint64", ByteSize: 8}}}
	overflow := &dwarf.ArrayType{Type: byteType, Count: math.MaxInt64}
	for _, target := range []dwarf.Type{cyclic, overflow} {
		w := typedDataWalker{references: map[dwarf.Type]bool{}}
		ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: target}
		require.Nil(t, w.pointerTarget(ptr))
	}
}
