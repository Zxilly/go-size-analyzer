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

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type ElfWrapper struct {
	file *elf.File
}

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
	case strings.HasSuffix(s.Name, "bss") || strings.HasSuffix(s.Name, "data"):
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
	ef := e.file
	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		data := make([]byte, size)
		if prog.Vaddr <= addr && addr+size-1 <= prog.Vaddr+prog.Filesz-1 {
			if _, err := prog.ReadAt(data, int64(addr-prog.Vaddr)); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, ErrAddrNotFound
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
	}
	return ""
}
