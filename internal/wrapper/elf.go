package wrapper

import (
	"debug/elf"
	"errors"
	"fmt"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type ElfWrapper struct {
	file *elf.File
}

func (*ElfWrapper) PclntabSections() []string {
	return []string{".gopclntab", ".data.rel.ro.gopclntab", ".data.rel.ro"}
}

func (e *ElfWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType) error) error {
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

	for _, s := range symbols {
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
		typ := entity.AddrTypeUnknown
		switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
		case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
			typ = entity.AddrTypeText
		case elf.SHF_ALLOC:
			typ = entity.AddrTypeData
		default:
			continue // wtf?
		}

		err = marker(s.Name, s.Value, s.Size, typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *ElfWrapper) LoadSections() map[string]*entity.Section {
	ret := make(map[string]*entity.Section)
	for _, section := range e.file.Sections {
		// not exist in binary
		if section.Type == elf.SHT_NULL || section.Size == 0 {
			continue
		}

		// check if debug
		d := strings.HasPrefix(section.Name, ".debug_") || strings.HasPrefix(section.Name, ".zdebug_")

		if _, ok := ret[section.Name]; ok {
			panic(fmt.Sprintf("section %s already exists", section.Name))
		}

		if section.Type == elf.SHT_NOBITS {
			// seems like .bss section
			ret[section.Name] = &entity.Section{
				Name:         section.Name,
				Addr:         section.Addr,
				AddrEnd:      section.Addr + section.Size,
				OnlyInMemory: true,
				Debug:        d,
			}
			continue
		}

		ret[section.Name] = &entity.Section{
			Name:         section.Name,
			Size:         section.Size,
			FileSize:     section.FileSize,
			Offset:       section.Offset,
			End:          section.Offset + section.FileSize,
			Addr:         section.Addr,
			AddrEnd:      section.Addr + section.Size,
			OnlyInMemory: false,
			Debug:        d,
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
	return nil, fmt.Errorf("address not found")
}

func (e *ElfWrapper) Text() (textStart uint64, text []byte, err error) {
	sect := e.file.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return textStart, text, err
}

func (e *ElfWrapper) GoArch() string {
	switch e.file.Machine {
	// case elf.EM_386:
	// 	return "386"
	case elf.EM_X86_64:
		return "amd64"
		// case elf.EM_ARM:
		// 	return "arm"
		// case elf.EM_AARCH64:
		// 	return "arm64"
		// case elf.EM_PPC64:
		// 	if e.file.ByteOrder == binary.LittleEndian {
		// 		return "ppc64le"
		// 	}
		// 	return "ppc64"
		// case elf.EM_S390:
		// 	return "s390x"
	}
	return ""
}
