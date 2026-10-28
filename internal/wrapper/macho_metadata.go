package wrapper

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (m *MachoWrapper) loadFileMetadata(r metadataReader) (FileMetadata, error) {
	var result FileMetadata
	if m == nil || m.file == nil {
		return result, errors.New("missing Mach-O metadata header")
	}
	header, err := r.read(0, 28)
	if err != nil {
		return result, err
	}
	var order binary.ByteOrder = binary.LittleEndian
	magic := order.Uint32(header)
	if magic != 0xfeedface && magic != 0xfeedfacf {
		order = binary.BigEndian
		magic = order.Uint32(header)
	}
	if magic != 0xfeedface && magic != 0xfeedfacf {
		return result, errors.New("invalid Mach-O metadata magic")
	}
	is64 := magic == 0xfeedfacf
	headerSize := uint64(28)
	if is64 {
		headerSize = 32
	}
	count, commandSize := order.Uint32(header[16:]), uint64(order.Uint32(header[20:]))
	if uint64(count) > commandSize/8 {
		return result, errors.New("Mach-O command count exceeds command area")
	}
	commands, err := r.read(headerSize, commandSize)
	if err != nil {
		return result, err
	}
	if err := r.add(&result, 0, headerSize+commandSize, "macho-load-commands"); err != nil {
		return result, err
	}
	for _, s := range m.file.Sections {
		if !machoSectionShouldIgnore(s) && (strings.HasPrefix(s.Name, "__go_") || s.Name == "__gopclntab" || s.Name == "__gosymtab" || s.Name == "__typelink" || s.Name == "__itablink") {
			if err := r.add(&result, uint64(s.Offset), s.Size, "macho-metadata:"+s.Name); err != nil {
				return result, err
			}
		}
	}
	pos := uint64(0)
	for i := uint32(0); i < count; i++ {
		if pos > commandSize || commandSize-pos < 8 {
			return result, errors.New("truncated Mach-O load command")
		}
		cmd, length := order.Uint32(commands[pos:]), uint64(order.Uint32(commands[pos+4:]))
		if length < 8 || length > commandSize-pos || length%4 != 0 {
			return result, fmt.Errorf("invalid Mach-O load command size %d", length)
		}
		b := commands[pos : pos+length]
		pos += length
		minimum := func(n int) error {
			if len(b) < n {
				return fmt.Errorf("truncated Mach-O load command %#x", cmd)
			}
			return nil
		}
		addPair := func(offset int, stride uint64, source string) error {
			n, err := metadataProduct(uint64(order.Uint32(b[offset+4:])), stride)
			if err != nil {
				return err
			}
			return r.add(&result, uint64(order.Uint32(b[offset:])), n, source)
		}
		switch cmd {
		case 1, 0x19: // LC_SEGMENT, LC_SEGMENT_64
			need := 56
			if cmd == 0x19 {
				need = 72
			}
			if err := minimum(need); err != nil {
				return result, err
			}
			if string(bytes.TrimRight(b[8:24], "\x00")) == "__LINKEDIT" {
				off, size := uint64(order.Uint32(b[32:])), uint64(order.Uint32(b[36:]))
				if cmd == 0x19 {
					off, size = order.Uint64(b[40:]), order.Uint64(b[48:])
				}
				// The segment also contains alignment gaps. Only the explicit
				// table and payload commands below contribute recognized bytes.
				if err := r.check(off, size); err != nil {
					return result, err
				}
			}
		case 2: // LC_SYMTAB
			if err := minimum(24); err != nil {
				return result, err
			}
			if err := m.readSymbolMetadata(r, &result, order, is64, b); err != nil {
				return result, err
			}
		case 0xb: // LC_DYSYMTAB
			if err := minimum(80); err != nil {
				return result, err
			}
			moduleSize := uint64(52)
			if is64 {
				moduleSize = 56
			}
			for _, table := range []struct {
				offset int
				stride uint64
			}{{32, 8}, {40, moduleSize}, {48, 4}, {56, 4}, {64, 8}, {72, 8}} {
				if err := addPair(table.offset, table.stride, "macho-dynamic-symbol-table"); err != nil {
					return result, err
				}
			}
		case 0x22, 0x80000022: // LC_DYLD_INFO[_ONLY]
			if err := minimum(48); err != nil {
				return result, err
			}
			for offset := 8; offset < 48; offset += 8 {
				if err := addPair(offset, 1, "macho-dyld-info"); err != nil {
					return result, err
				}
			}
		case 3, 0x1d, 0x1e, 0x26, 0x29, 0x2b, 0x2e, 0x80000033, 0x80000034:
			// Symbol segments and linkedit_data_command payloads: code signature,
			// split info, function starts, data-in-code, DRS, optimization hints,
			// export trie and chained fixups. Their payloads need not be sections.
			if err := minimum(16); err != nil {
				return result, err
			}
			if err := addPair(8, 1, "macho-linkedit-data"); err != nil {
				return result, err
			}
		case 0x16: // LC_TWOLEVEL_HINTS
			if err := minimum(16); err != nil {
				return result, err
			}
			if err := addPair(8, 4, "macho-twolevel-hints"); err != nil {
				return result, err
			}
		default:
			// Unknown payloads have no verified physical range to claim.
		}
	}
	if pos != commandSize {
		return result, errors.New("Mach-O load commands do not fill declared area")
	}
	return result, nil
}

func (m *MachoWrapper) readSymbolMetadata(r metadataReader, result *FileMetadata, order binary.ByteOrder, is64 bool, command []byte) error {
	stride := uint64(12)
	if is64 {
		stride = 16
	}
	off := uint64(order.Uint32(command[8:]))
	count := uint64(order.Uint32(command[12:]))
	strOff, strSize := uint64(order.Uint32(command[16:])), uint64(order.Uint32(command[20:]))
	tableSize, err := metadataProduct(count, stride)
	if err != nil {
		return err
	}
	if err := r.add(result, off, tableSize, "macho-symbol-table"); err != nil {
		return err
	}
	if err := r.add(result, strOff, strSize, "macho-string-table"); err != nil {
		return err
	}
	symbols, err := r.read(off, tableSize)
	if err != nil {
		return err
	}
	names, err := r.read(strOff, strSize)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		b := symbols[i*stride : (i+1)*stride]
		// Only N_SECT symbols define an address; N_STAB and N_INDR do not.
		if b[4]&0xe0 != 0 || b[4]&0x0e != 0x0e || b[5] == 0 || int(b[5]) > len(m.file.Sections) {
			continue
		}
		target := m.file.Sections[b[5]-1]
		value := uint64(order.Uint32(b[8:]))
		if is64 {
			value = order.Uint64(b[8:])
		}
		if machoSectionShouldIgnore(target) || metadataDebugSection(target.Name) || value < target.Addr || value-target.Addr >= target.Size {
			continue
		}
		var typ entity.AddrType
		switch machoSectionType(target) {
		case entity.SectionContentText:
			typ = entity.AddrTypeText
		case entity.SectionContentData:
			typ = entity.AddrTypeData
		default:
			continue
		}
		if err := r.check(uint64(target.Offset), target.Size); err != nil {
			return err
		}
		nameOff := uint64(order.Uint32(b))
		if nameOff == 0 {
			continue
		}
		name, nameSize, err := metadataName(names, nameOff)
		if err != nil {
			return err
		}
		// Match go-macho's Go symbol normalization, retaining the full on-disk
		// name in the metadata range (including the removed leading underscore).
		if strings.HasPrefix(name, "_") && strings.Contains(name, ".") {
			name = name[1:]
		}
		if !metadataSymbolName(name) {
			continue
		}
		result.Symbols = append(result.Symbols, SymbolMetadata{Name: name, Addr: value, Type: typ, Ranges: []MetadataRange{
			metadataRange(off+i*stride, stride, "symbol_table"),
			metadataRange(strOff+nameOff, nameSize, "symbol_name"),
		}})
	}
	return nil
}
