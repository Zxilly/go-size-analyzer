package wrapper

import (
	"cmp"
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// relocEntry is a single R_*_RELATIVE relocation: the pointer at offset
// should be replaced with addend (for RELA) or left as-is (addend==0, for REL).
type relocEntry struct {
	offset uint64 // virtual address where the pointer is stored
	addend uint64 // resolved value to patch in (0 means use disk value)
}

type ElfWrapper struct {
	file      *elf.File
	relocs    []relocEntry // sorted by offset; built lazily
	relocOnce sync.Once
}

func (e *ElfWrapper) buildRelocs() {
	e.relocOnce.Do(func() {
		f := e.file
		is32 := f.Class == elf.ELFCLASS32
		order := f.ByteOrder

		for _, s := range f.Sections {
			if s.Type != elf.SHT_RELA {
				continue
			}
			data, err := s.Data()
			if err != nil {
				continue
			}
			if is32 {
				relTypeRelative := uint32(elf.R_386_RELATIVE)
				for i := 0; i+12 <= len(data); i += 12 {
					info := order.Uint32(data[i+4:])
					if info == relTypeRelative {
						e.relocs = append(e.relocs, relocEntry{
							offset: uint64(order.Uint32(data[i:])),
							addend: uint64(int64(int32(order.Uint32(data[i+8:])))),
						})
					}
				}
			} else {
				var relTypeRelative uint32
				switch f.Machine {
				case elf.EM_X86_64:
					relTypeRelative = uint32(elf.R_X86_64_RELATIVE)
				case elf.EM_AARCH64:
					relTypeRelative = uint32(elf.R_AARCH64_RELATIVE)
				default:
					continue
				}
				for i := 0; i+24 <= len(data); i += 24 {
					info := order.Uint64(data[i+8:])
					if uint32(info&0xffffffff) == relTypeRelative {
						e.relocs = append(e.relocs, relocEntry{
							offset: order.Uint64(data[i:]),
							addend: order.Uint64(data[i+16:]),
						})
					}
				}
			}
		}
		slices.SortFunc(e.relocs, func(a, b relocEntry) int {
			return cmp.Compare(a.offset, b.offset)
		})
	})
}

var _ RawFileWrapper = (*ElfWrapper)(nil)

func (e *ElfWrapper) DWARF() (*dwarf.Data, error) {
	return e.file.DWARF()
}

func (e *ElfWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType), goSCb func(addr, size uint64)) error {
	symbols, err := e.file.Symbols()
	if err != nil {
		if errors.Is(err, elf.ErrNoSymbols) {
			return ErrNoSymbolTable
		}
		return err
	}

	keep := make([]elf.Symbol, 0)
	for _, s := range symbols {
		if ignoreSymbols.Contains(s.Name) {
			continue
		}
		keep = append(keep, s)
	}
	symbols = keep

	slices.SortFunc(symbols, func(a, b elf.Symbol) int {
		return cmp.Compare(a.Value, b.Value)
	})

	goStringBase := uint64(0)

	for _, s := range symbols {
		if s.Name == GoStringSymbol {
			goStringBase = s.Value
			continue
		}
		if goStringBase != 0 {
			goSCb(goStringBase, s.Value-goStringBase)
			if marker == nil {
				return nil
			}
			continue
		}

		if marker == nil {
			continue
		}

		switch s.Section {
		case elf.SHN_UNDEF, elf.SHN_ABS, elf.SHN_COMMON:
			continue // not addr, skip
		default:
		}

		if s.Size == 0 {
			// nothing to do
			continue
		}

		i := int(s.Section)
		if i < 0 || i >= len(e.file.Sections) {
			// just ignore, example: we met go.go
			continue
		}
		sect := e.file.Sections[i]
		var typ entity.AddrType
		switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
		case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
			typ = entity.AddrTypeText
		case elf.SHF_ALLOC:
			typ = entity.AddrTypeData
		default:
			continue // wtf?
		}

		marker(s.Name, s.Value, s.Size, typ)
	}

	return nil
}

func elfSectionType(s *elf.Section) entity.SectionContentType {
	switch {
	case s.Name == ".text":
		return entity.SectionContentText
	// Contains "data" covers .data, .rodata, .noptrdata and PIE's
	// .data.rel.ro / .data.rel.ro.local — all hold runtime-visible data
	// (type descriptors, string literals, etc.) that symbol analysis
	// legitimately indexes into.
	case strings.HasSuffix(s.Name, "bss") || strings.Contains(s.Name, "data"):
		return entity.SectionContentData
	default:
		return entity.SectionContentOther
	}
}

func (e *ElfWrapper) LoadSections() *entity.Store {
	ret := entity.NewStore()

	for _, s := range e.file.Sections {
		// not exist in binary
		if s.Type == elf.SHT_NULL || s.Size == 0 {
			continue
		}

		// check if debug
		d := strings.HasPrefix(s.Name, ".debug_") || strings.HasPrefix(s.Name, ".zdebug_")

		if _, ok := ret.Sections[s.Name]; ok {
			panic(fmt.Errorf("section %s already exists", s.Name))
		}

		if s.Type == elf.SHT_NOBITS {
			// seems like .bss section
			ret.Sections[s.Name] = &entity.Section{
				Name:         s.Name,
				Size:         s.Size,
				Addr:         s.Addr,
				AddrEnd:      s.Addr + s.Size,
				OnlyInMemory: true,
				Debug:        d,
				ContentType:  elfSectionType(s),
			}
			continue
		}

		ret.Sections[s.Name] = &entity.Section{
			Name:         s.Name,
			Size:         s.Size,
			FileSize:     s.FileSize,
			Offset:       s.Offset,
			End:          s.Offset + s.FileSize,
			Addr:         s.Addr,
			AddrEnd:      s.Addr + s.Size,
			OnlyInMemory: false,
			Debug:        d,
			ContentType:  elfSectionType(s),
		}
	}
	return ret
}

func (e *ElfWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	ef := e.file
	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		if prog.Vaddr <= addr && addr+size-1 <= prog.Vaddr+prog.Filesz-1 {
			data := make([]byte, size)
			if _, err := prog.ReadAt(data, int64(addr-prog.Vaddr)); err != nil {
				return nil, err
			}
			e.applyRelocations(data, addr)
			return data, nil
		}
	}
	return nil, ErrAddrNotFound
}

// applyRelocations patches pointer-sized fields in data (read from baseAddr)
// with resolved PIE relocation addends from .rela.dyn.
// Only R_*_RELATIVE entries are applied; non-pointer bytes are untouched.
func (e *ElfWrapper) applyRelocations(data []byte, baseAddr uint64) {
	e.buildRelocs()
	if len(e.relocs) == 0 {
		return
	}

	var ptrSize int
	if e.file.Class == elf.ELFCLASS64 {
		ptrSize = 8
	} else {
		ptrSize = 4
	}
	order := e.file.ByteOrder
	end := baseAddr + uint64(len(data))

	// Binary search for the first relocation entry >= baseAddr.
	i, _ := slices.BinarySearchFunc(e.relocs, baseAddr, func(r relocEntry, target uint64) int {
		return cmp.Compare(r.offset, target)
	})
	for ; i < len(e.relocs) && e.relocs[i].offset < end; i++ {
		r := e.relocs[i]
		if r.offset+uint64(ptrSize) > end {
			break
		}
		off := r.offset - baseAddr
		if r.addend != 0 {
			if ptrSize == 8 {
				order.PutUint64(data[off:], r.addend)
			} else {
				order.PutUint32(data[off:], uint32(r.addend))
			}
		}
	}
}

func (e *ElfWrapper) Text() (textStart uint64, text []byte, err error) {
	sect := e.file.Section(".text")
	if sect == nil {
		return 0, nil, errors.New("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return textStart, text, err
}

func (e *ElfWrapper) GoArch() string {
	switch e.file.Machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	case elf.EM_PPC64:
		if e.file.ByteOrder == binary.LittleEndian {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_S390:
		return "s390x"
	default:
		return ""
	}
}
