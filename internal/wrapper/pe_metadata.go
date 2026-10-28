package wrapper

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (p *PeWrapper) loadFileMetadata(r metadataReader) (FileMetadata, error) {
	var result FileMetadata
	if p == nil || p.file == nil {
		return result, errors.New("missing PE metadata header")
	}
	off, count := uint64(p.file.PointerToSymbolTable), uint64(p.file.NumberOfSymbols)
	if off == 0 && count == 0 {
		return result, nil
	}
	if off == 0 {
		return result, errors.New("PE symbol table has no file offset")
	}
	const stride = uint64(18)
	tableSize, err := metadataProduct(count, stride)
	if err != nil {
		return result, err
	}
	if err := r.add(&result, off, tableSize, "pe-coff-symbol-table"); err != nil {
		return result, err
	}
	strOff := off + tableSize // table bounds were checked before addition.
	header, err := r.read(strOff, 4)
	if err != nil {
		return result, err
	}
	strSize := uint64(binary.LittleEndian.Uint32(header))
	if strSize < 4 {
		return result, fmt.Errorf("invalid PE string table size %d", strSize)
	}
	if err := r.add(&result, strOff, strSize, "pe-coff-string-table"); err != nil {
		return result, err
	}
	symbols, err := r.read(off, tableSize)
	if err != nil {
		return result, err
	}
	names, err := r.read(strOff, strSize)
	if err != nil {
		return result, err
	}
	for i := uint64(0); i < count; {
		b := symbols[i*stride : (i+1)*stride]
		aux := uint64(b[17])
		if aux >= count-i {
			return result, errors.New("PE auxiliary symbol records exceed table")
		}
		recordOff, recordSize := off+i*stride, (1+aux)*stride
		i += 1 + aux
		section := int16(binary.LittleEndian.Uint16(b[12:]))
		if section <= 0 || int(section) > len(p.file.Sections) {
			continue
		}
		// File, section, label and scope records do not define an owned object.
		if b[16] != 2 && b[16] != 3 && b[16] != 105 {
			continue
		}
		target := p.file.Sections[section-1]
		value := uint64(binary.LittleEndian.Uint32(b[8:]))
		if target.Offset == 0 || value >= uint64(target.Size) || metadataDebugSection(target.Name) || target.Characteristics&pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA != 0 {
			continue
		}
		var typ entity.AddrType
		switch {
		case target.Characteristics&pe.IMAGE_SCN_CNT_CODE != 0:
			typ = entity.AddrTypeText
		case target.Characteristics&pe.IMAGE_SCN_CNT_INITIALIZED_DATA != 0:
			typ = entity.AddrTypeData
		default:
			continue
		}
		if err := r.check(uint64(target.Offset), uint64(target.Size)); err != nil {
			return result, err
		}
		nameLen := bytes.IndexByte(b[:8], 0)
		if nameLen < 0 {
			nameLen = 8
		}
		name := string(b[:nameLen])
		ranges := []MetadataRange{metadataRange(recordOff, recordSize, "symbol_table")}
		if binary.LittleEndian.Uint32(b) == 0 {
			nameOff := uint64(binary.LittleEndian.Uint32(b[4:]))
			if nameOff < 4 {
				return result, errors.New("PE symbol name points into string table header")
			}
			var nameSize uint64
			name, nameSize, err = metadataName(names, nameOff)
			if err != nil {
				return result, err
			}
			ranges = append(ranges, metadataRange(strOff+nameOff, nameSize, "symbol_name"))
		}
		if !metadataSymbolName(name) || (b[16] == 3 && name == target.Name) {
			continue
		}
		base := uint64(target.VirtualAddress)
		if base > math.MaxUint64-p.imageBase || value > math.MaxUint64-(p.imageBase+base) {
			return result, errors.New("PE symbol address overflows")
		}
		result.Symbols = append(result.Symbols, SymbolMetadata{Name: name, Addr: p.imageBase + base + value, Type: typ, Ranges: ranges})
	}
	return result, nil
}
