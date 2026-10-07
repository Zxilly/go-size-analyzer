package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// MemoryReader decodes values in the analyzed binary's byte order.
type MemoryReader struct {
	Read      func(addr, size uint64) ([]byte, error)
	ByteOrder binary.ByteOrder
}

func readUintTo64(data []byte, order binary.ByteOrder) (uint64, error) {
	if order == nil {
		return 0, errors.New("missing memory byte order")
	}
	switch len(data) {
	case 4:
		return uint64(order.Uint32(data)), nil
	case 8:
		return order.Uint64(data), nil
	default:
		return 0, fmt.Errorf("unexpected integer size: %d", len(data))
	}
}

// Validate the complete header before asking the wrapper to allocate/read it.
func checkHeader(typ *dwarf.StructType, fields ...fieldPattern) error {
	if err := checkField(typ, fields...); err != nil {
		return err
	}
	if typ.Size() < 0 {
		return errors.New("negative header size")
	}
	var end int64
	var width int64
	for i, field := range typ.Field {
		size := field.Type.Size()
		if size != 4 && size != 8 {
			return fmt.Errorf("field %d has unsupported width %d", i, size)
		}
		if i == 0 {
			width = size
		}
		if size != width {
			return fmt.Errorf("field %d width differs from pointer width", i)
		}
		if field.ByteOffset != int64(i)*width || field.BitSize != 0 || field.BitOffset != 0 {
			return fmt.Errorf("field %d has unsupported Go header layout", i)
		}
		if field.ByteOffset < end || field.ByteOffset > typ.Size() || size > typ.Size()-field.ByteOffset {
			return fmt.Errorf("field %d lies outside header or overlaps another field", i)
		}
		end = field.ByteOffset + size
	}
	return nil
}

func (m MemoryReader) readExact(addr, size uint64) ([]byte, error) {
	if m.Read == nil || m.ByteOrder == nil {
		return nil, errors.New("incomplete memory reader")
	}
	if size > ^uint64(0)-addr {
		return nil, errors.New("memory address range overflow")
	}
	data, err := m.Read(addr, size)
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) < size {
		return nil, fmt.Errorf("short memory read: got %d bytes, want %d", len(data), size)
	}
	return data[:size], nil
}

func readHeader(typ *dwarf.StructType, addr uint64, m MemoryReader) ([]uint64, error) {
	// Only the validated fields are needed, so ignore any trailing DWARF padding.
	last := typ.Field[len(typ.Field)-1]
	data, err := m.readExact(addr, uint64(last.ByteOffset+last.Type.Size()))
	if err != nil {
		return nil, err
	}
	values := make([]uint64, len(typ.Field))
	for i, field := range typ.Field {
		values[i], err = readUintTo64(data[field.ByteOffset:field.ByteOffset+field.Type.Size()], m.ByteOrder)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func signedLength(value uint64, width int64) (uint64, error) {
	if width == 4 && int32(value) < 0 || width == 8 && int64(value) < 0 {
		return 0, errors.New("negative header length")
	}
	return value, nil
}

func readString(typ *dwarf.StructType, addr uint64, m MemoryReader) (dataAddr uint64, size uint64, err error) {
	if err := checkHeader(typ, fieldPattern{"str", "*uint8"}, fieldPattern{"len", "int"}); err != nil {
		return 0, 0, err
	}
	values, err := readHeader(typ, addr, m)
	if errors.Is(err, wrapper.ErrAddrNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	length, err := signedLength(values[1], typ.Field[1].Type.Size())
	if err != nil {
		return 0, 0, err
	}
	if length > ^uint64(0)-values[0] {
		return 0, 0, errors.New("string address range overflow")
	}
	return values[0], length, nil
}

func readSlice(typ *dwarf.StructType, addr uint64, m MemoryReader, memberTyp string) (dataAddr uint64, size uint64, err error) {
	if err := checkHeader(typ, fieldPattern{"array", memberTyp}, fieldPattern{"len", "int"}, fieldPattern{"cap", "int"}); err != nil {
		return 0, 0, err
	}
	values, err := readHeader(typ, addr, m)
	if errors.Is(err, wrapper.ErrAddrNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	length, err := signedLength(values[1], typ.Field[1].Type.Size())
	if err != nil {
		return 0, 0, err
	}
	capacity, err := signedLength(values[2], typ.Field[2].Type.Size())
	if err != nil {
		return 0, 0, err
	}
	if length != capacity {
		return 0, 0, fmt.Errorf("byte slice len(%d) != cap(%d)", length, capacity)
	}
	return values[0], length, nil
}

func readEmbedFS(typ *dwarf.StructType, typAddr uint64, readMemory MemoryReader) ([]Content, error) {
	err := checkHeader(typ, fieldPattern{"files", "*struct []embed.file"})
	if err != nil {
		return nil, err
	}

	// read ptr
	ptrSize := typ.Field[0].Type.Size()
	values, err := readHeader(typ, typAddr, readMemory)
	if err != nil {
		if errors.Is(err, wrapper.ErrAddrNotFound) {
			// a memory only variable
			return nil, nil
		}

		return nil, err
	}

	ptr := values[0]

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

	if filesType.Field[0].Type.Size() != ptrSize {
		return nil, errors.New("embed.FS file pointer width differs from header width")
	}

	// read files

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
	overflow, readSize := bits.Mul64(filesLen, fileStructSize)
	if overflow != 0 {
		return nil, fmt.Errorf("embed.FS files length overflow: %d entries of %d bytes", filesLen, fileStructSize)
	}
	data, err := readMemory.readExact(filesAddr, readSize)
	if err != nil {
		return nil, err
	}

	contents := make([]Content, 0, filesLen*3) // name, data, hash; fileStructSize >= 3 bounds filesLen
	for i := range filesLen {
		offset := int64(i * fileStructSize)

		nameAddr, err := readUintTo64(data[offset:offset+ptrSize], readMemory.ByteOrder)
		if err != nil {
			return nil, err
		}
		nameLen, err := readUintTo64(data[offset+ptrSize:offset+ptrSize*2], readMemory.ByteOrder)
		if err != nil {
			return nil, err
		}

		nameLen, err = signedLength(nameLen, ptrSize)
		if err != nil {
			return nil, err
		}
		nameData, err := readMemory.readExact(nameAddr, nameLen)
		if err != nil {
			return nil, err
		}
		name := utils.Deduplicate(fmt.Sprintf("embed:%s", string(nameData)))

		dataAddr, err := readUintTo64(data[offset+ptrSize*2:offset+ptrSize*3], readMemory.ByteOrder)
		if err != nil {
			return nil, err
		}
		dataLen, err := readUintTo64(data[offset+ptrSize*3:offset+ptrSize*4], readMemory.ByteOrder)
		if err != nil {
			return nil, err
		}

		dataLen, err = signedLength(dataLen, ptrSize)
		if err != nil {
			return nil, err
		}
		if dataLen > ^uint64(0)-dataAddr {
			return nil, errors.New("embed.FS data address range overflow")
		}

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
