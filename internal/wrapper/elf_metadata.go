package wrapper

import (
	"debug/elf"
	"errors"
	"fmt"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (e *ElfWrapper) loadFileMetadata(r metadataReader) (FileMetadata, error) {
	var result FileMetadata
	if e == nil || e.file == nil || e.file.ByteOrder == nil {
		return result, errors.New("missing ELF metadata header")
	}
	entrySize := uint64(24)
	switch e.file.Class {
	case elf.ELFCLASS32:
		entrySize = 16
	case elf.ELFCLASS64:
	default:
		return result, fmt.Errorf("unsupported ELF symbol class %v", e.file.Class)
	}
	for index, s := range e.file.Sections {
		if s.Type == elf.SHT_NOBITS || s.Type == elf.SHT_NULL {
			continue
		}
		if s.Type == elf.SHT_STRTAB || s.Type == elf.SHT_SYMTAB_SHNDX || elfCompilerMetadata(s.Name) {
			if err := r.add(&result, s.Offset, s.FileSize, "elf-metadata:"+s.Name); err != nil {
				return result, err
			}
		}
		if s.Type != elf.SHT_SYMTAB && s.Type != elf.SHT_DYNSYM {
			continue
		}
		if err := r.add(&result, s.Offset, s.FileSize, "elf-symbol-table"); err != nil {
			return result, err
		}
		// Compressed offsets refer to the decoded stream, not file bytes.
		if s.FileSize == 0 || s.Flags&elf.SHF_COMPRESSED != 0 {
			continue
		}
		if s.Entsize != entrySize || s.FileSize%entrySize != 0 {
			return result, fmt.Errorf("invalid ELF symbol table entry size in %s", s.Name)
		}
		if int64(s.Link) >= int64(len(e.file.Sections)) {
			return result, fmt.Errorf("invalid ELF symbol string table link %d", s.Link)
		}
		strtab := e.file.Sections[s.Link]
		if strtab.Type != elf.SHT_STRTAB {
			return result, errors.New("ELF symbol table links to a non-string section")
		}
		if err := r.add(&result, strtab.Offset, strtab.FileSize, "elf-string-table"); err != nil {
			return result, err
		}
		if strtab.Flags&elf.SHF_COMPRESSED != 0 {
			continue
		}
		symbols, err := r.read(s.Offset, s.FileSize)
		if err != nil {
			return result, err
		}
		names, err := r.read(strtab.Offset, strtab.FileSize)
		if err != nil {
			return result, err
		}
		var extended []byte
		for _, candidate := range e.file.Sections {
			if candidate.Type == elf.SHT_SYMTAB_SHNDX && uint64(candidate.Link) == uint64(index) {
				if candidate.Flags&elf.SHF_COMPRESSED != 0 || candidate.FileSize/4 != s.FileSize/entrySize || candidate.FileSize%4 != 0 {
					return result, errors.New("invalid ELF extended symbol section indices")
				}
				extended, err = r.read(candidate.Offset, candidate.FileSize)
				if err != nil {
					return result, err
				}
				break
			}
		}
		order := e.file.ByteOrder
		for off := entrySize; off < uint64(len(symbols)); off += entrySize {
			b := symbols[off : off+entrySize]
			nameOff := uint64(order.Uint32(b))
			var value uint64
			var section uint32
			var info byte
			if entrySize == 16 {
				value, info, section = uint64(order.Uint32(b[4:])), b[12], uint32(order.Uint16(b[14:]))
			} else {
				value, info, section = order.Uint64(b[8:]), b[4], uint32(order.Uint16(b[6:]))
			}
			if section == uint32(elf.SHN_XINDEX) {
				if extended == nil {
					return result, errors.New("missing ELF extended symbol section indices")
				}
				section = order.Uint32(extended[off/entrySize*4:])
			} else if section >= uint32(elf.SHN_LORESERVE) {
				continue
			}
			switch elf.ST_TYPE(info) {
			case elf.STT_NOTYPE, elf.STT_OBJECT, elf.STT_FUNC, elf.STT_GNU_IFUNC:
			default:
				continue
			}
			if section == 0 || uint64(section) >= uint64(len(e.file.Sections)) {
				continue
			}
			target := e.file.Sections[section]
			if target.Type == elf.SHT_NOBITS || target.Flags&elf.SHF_COMPRESSED != 0 || metadataDebugSection(target.Name) || value < target.Addr || value-target.Addr >= target.FileSize {
				continue
			}
			var typ entity.AddrType
			switch target.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
			case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
				typ = entity.AddrTypeText
			case elf.SHF_ALLOC, elf.SHF_ALLOC | elf.SHF_WRITE:
				typ = entity.AddrTypeData
			default:
				continue
			}
			if err := r.check(target.Offset, target.FileSize); err != nil {
				return result, err
			}
			name, nameSize, err := metadataName(names, nameOff)
			if err != nil {
				return result, err
			}
			if !metadataSymbolName(name) {
				continue
			}
			result.Symbols = append(result.Symbols, SymbolMetadata{Name: name, Addr: value, Type: typ, Ranges: []MetadataRange{
				metadataRange(s.Offset+off, entrySize, "symbol_table"),
				metadataRange(strtab.Offset+nameOff, nameSize, "symbol_name"),
			}})
		}
	}
	return result, nil
}

func elfCompilerMetadata(name string) bool {
	return strings.HasPrefix(name, ".go.") || name == ".gopclntab" || name == ".gosymtab" || name == ".typelink" || name == ".itablink"
}
