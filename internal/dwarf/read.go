package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"math"
)

func readUintTo64(data []byte) uint64 {
	if len(data) == 4 {
		return uint64(binary.LittleEndian.Uint32(data))
	} else if len(data) == 8 {
		return binary.LittleEndian.Uint64(data)
	} else {
		panic(fmt.Sprintf("unexpected size: %d", len(data)))
	}
}

func readIntTo64(data []byte) int64 {
	if len(data) == 4 {
		return int64(binary.LittleEndian.Uint32(data))
	} else if len(data) == 8 {
		return int64(binary.LittleEndian.Uint64(data))
	} else {
		panic(fmt.Sprintf("unexpected size: %d", len(data)))
	}
}

type MemoryReader func(addr, size uint64) ([]byte, error)

func readString(structTyp *dwarf.StructType, readMemory MemoryReader) (uint64, uint64, error) {
	if len(structTyp.Field) != 2 {
		return 0, 0, fmt.Errorf("string struct has %d fields", len(structTyp.Field))
	}

	// check field
	if structTyp.Field[0].Name != "str" && structTyp.Field[1].Name != "len" {
		return 0, 0, fmt.Errorf("string struct has wrong field name")
	}

	ptrSize := structTyp.Field[0].Type.Size()
	lenOffset := structTyp.Field[1].ByteOffset

	readSize := structTyp.Size()

	data, err := readMemory(math.MaxUint64, uint64(readSize))
	if err != nil {
		if errors.Is(err, wrapper.ErrAddrNotFound) {
			// a memory only variable
			return 0, 0, nil
		}

		return 0, 0, err
	}

	// read ptr
	ptr := readUintTo64(data[:ptrSize])
	strLen := readIntTo64(data[lenOffset:])

	return ptr, uint64(strLen), nil
}

func readSlice(typ *dwarf.StructType, readMemory MemoryReader) (uint64, uint64, error) {
	if len(typ.Field) != 3 {
		return 0, 0, fmt.Errorf("byte slice struct has %d fields", len(typ.Field))
	}

	// check field
	if typ.Field[0].Name != "array" && typ.Field[1].Name != "len" && typ.Field[2].Name != "cap" {
		return 0, 0, fmt.Errorf("byte slice struct has wrong field name")
	}

	ptrSize := typ.Field[0].Type.Size()
	lenOffset := typ.Field[1].ByteOffset
	lenSize := typ.Field[1].Type.Size()
	capOffset := typ.Field[2].ByteOffset
	capSize := typ.Field[2].Type.Size()

	readSize := typ.Size()

	data, err := readMemory(math.MaxUint64, uint64(readSize))
	if err != nil {
		if errors.Is(err, wrapper.ErrAddrNotFound) {
			// a memory only variable
			return 0, 0, nil
		}

		return 0, 0, err
	}

	// read ptr
	ptr := readUintTo64(data[:ptrSize])
	dataLen := readUintTo64(data[lenOffset : lenOffset+lenSize])
	dataCap := readUintTo64(data[capOffset : capOffset+capSize])

	if dataLen != dataCap {
		return 0, 0, fmt.Errorf("byte slice len(%d) != cap(%d)", dataLen, dataCap)
	}

	return ptr, dataLen, nil
}

func readEmbedFS(typ *dwarf.StructType, readMemory MemoryReader) ([]Content, error) {
	if len(typ.Field) != 1 {
		return nil, fmt.Errorf("embed fs struct has %d fields", len(typ.Field))
	}

	// check field
	if typ.Field[0].Name != "files" {
		return nil, fmt.Errorf("embed fs struct has wrong field name")
	}

	// read ptr
	ptrSize := typ.Field[0].Type.Size()
	data, err := readMemory(math.MaxUint64, uint64(ptrSize))
	if err != nil {
		if errors.Is(err, wrapper.ErrAddrNotFound) {
			// a memory only variable
			return nil, nil
		}

		return nil, err
	}

	ptr := readUintTo64(data)

	filesPtrType := typ.Field[0].Type.(*dwarf.PtrType)
	filesType := filesPtrType.Type.(*dwarf.StructType)

	filesAddr, filesLen, err := readSlice(filesType, func(addr, size uint64) ([]byte, error) {
		if addr == math.MaxUint64 {
			addr = ptr
		}
		return readMemory(addr, size)
	})

	if err != nil {
		return nil, err
	}

	if filesLen == 0 {
		// embed.FS contains no file? I'm not sure
		return nil, nil
	}

	// read files
	// tired of check size, just assume this not change

	// A file is a single file in the FS.
	// It implements fs.FileInfo and fs.DirEntry.
	// type file struct {
	// 	// The compiler knows the layout of this struct.
	// 	// See cmd/compile/internal/staticdata's WriteEmbed.
	// 	name string
	// 	data string
	// 	hash [16]byte // truncated SHA256 hash
	// }

	// read file struct for each
	fileStructSize := uint64(ptrSize)*2*2 + 16
	readSize := filesLen * fileStructSize
	data, err = readMemory(filesAddr, readSize)
	if err != nil {
		return nil, err
	}

	contents := make([]Content, 0, filesLen*3) // for name, data, hash
	for i := range filesLen {
		offset := int64(i * fileStructSize)

		nameAddr := readUintTo64(data[offset : offset+ptrSize])
		nameLen := readUintTo64(data[offset+ptrSize : offset+ptrSize*2])

		nameData, err := readMemory(nameAddr, nameLen)
		if err != nil {
			return nil, err
		}
		name := utils.Deduplicate(fmt.Sprintf("embed:%s", string(nameData)))

		dataAddr := readUintTo64(data[offset+ptrSize*2 : offset+ptrSize*3])
		dataLen := readUintTo64(data[offset+ptrSize*3 : offset+ptrSize*4])

		hashAddr := offset + ptrSize*4
		hashLen := 16

		contents = append(contents, Content{
			Name: utils.Deduplicate(fmt.Sprintf("%s.name", name)),
			Addr: nameAddr,
			Size: nameLen,
		}, Content{
			Name: utils.Deduplicate(fmt.Sprintf("%s.data", name)),
			Addr: dataAddr,
			Size: dataLen,
		}, Content{
			Name: utils.Deduplicate(fmt.Sprintf("%s.hash", name)),
			Addr: uint64(hashAddr),
			Size: uint64(hashLen),
		})
	}

	return contents, nil
}
