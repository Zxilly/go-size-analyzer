package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

func readUintTo64(data []byte) uint64 {
	switch len(data) {
	case 4:
		return uint64(binary.LittleEndian.Uint32(data))
	case 8:
		return binary.LittleEndian.Uint64(data)
	default:
		panic(fmt.Errorf("unexpected size: %d", len(data)))
	}
}

func readIntTo64(data []byte) int64 {
	switch len(data) {
	case 4:
		return int64(binary.LittleEndian.Uint32(data))
	case 8:
		return int64(binary.LittleEndian.Uint64(data))
	default:
		panic(fmt.Errorf("unexpected size: %d", len(data)))
	}
}

type MemoryReader func(addr, size uint64) ([]byte, error)

func readString(structTyp *dwarf.StructType, typAddr uint64, readMemory MemoryReader) (addr uint64, size uint64, err error) {
	err = checkField(structTyp, fieldPattern{"str", "*uint8"}, fieldPattern{"len", "int"})
	if err != nil {
		return 0, 0, err
	}

	ptrSize := structTyp.Field[0].Type.Size()
	lenOffset := structTyp.Field[1].ByteOffset

	readSize := structTyp.Size()

	data, err := readMemory(typAddr, uint64(readSize))
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

func readSlice(typ *dwarf.StructType, typAddr uint64, readMemory MemoryReader, memberTyp string) (addr uint64, size uint64, err error) {
	err = checkField(typ, fieldPattern{"array", memberTyp}, fieldPattern{"len", "int"}, fieldPattern{"cap", "int"})
	if err != nil {
		return 0, 0, err
	}

	ptrSize := typ.Field[0].Type.Size()
	lenOffset := typ.Field[1].ByteOffset
	lenSize := typ.Field[1].Type.Size()
	capOffset := typ.Field[2].ByteOffset
	capSize := typ.Field[2].Type.Size()

	readSize := typ.Size()

	data, err := readMemory(typAddr, uint64(readSize))
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

func readEmbedFS(typ *dwarf.StructType, typAddr uint64, readMemory MemoryReader) ([]Content, error) {
	err := checkField(typ, fieldPattern{"files", "*struct []embed.file"})
	if err != nil {
		return nil, err
	}

	// read ptr
	ptrSize := typ.Field[0].Type.Size()
	data, err := readMemory(typAddr, uint64(ptrSize))
	if err != nil {
		if errors.Is(err, wrapper.ErrAddrNotFound) {
			// a memory only variable
			return nil, nil
		}

		return nil, err
	}

	ptr := readUintTo64(data)

	filesPtrType, ok := typ.Field[0].Type.(*dwarf.PtrType)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", typ.Field[0].Type)
	}
	filesType, ok := filesPtrType.Type.(*dwarf.StructType)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", filesPtrType.Type)
	}

	filesAddr, filesLen, err := readSlice(filesType, ptr, readMemory, "*embed.file")
	if err != nil {
		return nil, err
	}

	if filesLen == 0 {
		// embed.FS contains no file? I'm not sure
		return nil, nil
	}

	// read files
	// tired of check size, just assume this not changes

	// a file struct is a single file in the FS.
	// it implements fs.FileInfo and fs.DirEntry.
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

		hashAddr := filesAddr + uint64(offset+ptrSize*4)
		hashLen := uint64(16)

		fileContent := make([]Content, 0, 3)
		if nameLen > 0 {
			fileContent = append(fileContent, Content{
				Name: utils.Deduplicate(fmt.Sprintf("%s.name", name)),
				Addr: nameAddr,
				Size: nameLen,
			})
		}
		if dataLen > 0 {
			fileContent = append(fileContent, Content{
				Name: utils.Deduplicate(fmt.Sprintf("%s.data", name)),
				Addr: dataAddr,
				Size: dataLen,
			})
		}
		fileContent = append(fileContent, Content{
			Name: utils.Deduplicate(fmt.Sprintf("%s.hash", name)),
			Addr: hashAddr,
			Size: hashLen,
		})

		contents = append(contents, fileContent...)
	}

	return contents, nil
}
