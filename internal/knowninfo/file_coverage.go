package knowninfo

import (
	"cmp"
	"encoding/binary"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

func recognizedSection(s *entity.Section) bool {
	if s.Debug {
		return true
	}
	switch s.Name {
	case ".symtab", ".strtab", ".shstrtab", ".dynsym", ".dynstr", ".dynamic", ".gnu.hash", ".hash", ".gosymtab", ".gopclntab", ".typelink", ".itablink", ".go.buildinfo", ".go.module", ".go.func", ".go.fipsinfo", ".eh_frame", ".eh_frame_hdr", ".pdata", ".xdata", ".idata", ".reloc", "__LINKEDIT":
		return true
	}
	return strings.HasPrefix(s.Name, ".note") || strings.HasPrefix(s.Name, ".rel.") || strings.HasPrefix(s.Name, ".rela.")
}

func (k *KnownInfo) BuildFileCoverage(reader io.ReaderAt, sections []*entity.Section, details bool) (*entity.FileCoverage, error) {
	ledger := entity.NewFileLedger(k.Size)
	var mappings []entity.FileMapping
	wasm, isWasm := k.Wrapper.(*wrapper.WasmWrapper)
	for _, s := range sections {
		if s.OnlyInMemory || s.VirtualSection || s.FileSize == 0 {
			continue
		}
		class := 0
		source := "section:" + s.Name
		if recognizedSection(s) || (isWasm && wasm.RecognizedSection(s.Name)) {
			class = 1
		}
		if err := ledger.Add(entity.FileClaim{FileRange: entity.FileRange{Offset: s.Offset, Size: s.FileSize}, Class: class, Source: source}); err != nil {
			return nil, err
		}
		if !isWasm && !s.Debug && s.ContentType != entity.SectionContentOther {
			mappings = append(mappings, entity.FileMapping{Addr: s.Addr, FileRange: entity.FileRange{Offset: s.Offset, Size: min(s.Size, s.FileSize)}})
		}
	}
	if isWasm {
		mappings = wasm.FileAddressMappings()
		for _, r := range wasm.FileMetadataRanges() {
			if err := ledger.Add(entity.FileClaim{FileRange: r, Class: 1, Source: "wasm-encoding"}); err != nil {
				return nil, err
			}
		}
	}
	if err := addFileHeaders(ledger, reader, k.Size); err != nil {
		return nil, err
	}
	err := k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		for addr := range p.OwnAddresses {
			if addr.Size == 0 || addr.Addr+addr.Size < addr.Addr {
				continue
			}
			candidates := mappings
			if isWasm {
				// WASM mappings are ordered and disjoint. Seek to the first
				// intersecting segment instead of scanning every sparse segment
				// for every function's metadata reference.
				start, _ := slices.BinarySearchFunc(mappings, addr.Addr, func(m entity.FileMapping, address uint64) int {
					if m.Addr+m.Size <= address {
						return -1
					}
					return 1
				})
				end, _ := slices.BinarySearchFunc(mappings, addr.Addr+addr.Size, func(m entity.FileMapping, address uint64) int {
					return cmp.Compare(m.Addr, address)
				})
				candidates = mappings[start:end]
			}
			for _, m := range candidates {
				lo, hi := max(addr.Addr, m.Addr), min(addr.Addr+addr.Size, m.Addr+m.Size)
				if lo >= hi {
					continue
				}
				if err := ledger.Add(entity.FileClaim{FileRange: entity.FileRange{Offset: m.Offset + lo - m.Addr, Size: hi - lo}, Class: 2, Owner: p.Name, Source: addr.SourceType}); err != nil {
					return err
				}
			}
		}
		if isWasm {
			for fn := range p.Functions {
				if r, ok := wasm.FunctionFileRange(fn.Addr, k.VersionFlag.Meq125); ok {
					if err := ledger.Add(entity.FileClaim{FileRange: r, Class: 2, Owner: p.Name, Source: entity.AddrSourceGoPclntab}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ledger.Finish(details), nil
}

func addFileHeaders(ledger *entity.FileLedger, r io.ReaderAt, size uint64) error {
	var h [64]byte
	n, _ := r.ReadAt(h[:], 0)
	add := func(off, n uint64, source string) error {
		return ledger.Add(entity.FileClaim{FileRange: entity.FileRange{Offset: off, Size: n}, Class: 1, Source: source})
	}
	if n >= 8 && string(h[:4]) == "\x00asm" {
		return add(0, 8, "wasm-header")
	}
	if n >= 52 && string(h[:4]) == "\x7fELF" {
		var order binary.ByteOrder = binary.LittleEndian
		if h[5] == 2 {
			order = binary.BigEndian
		}
		var ph, sh uint64
		var eh, pent, programCount, sent, sn uint16
		if h[4] == 2 && n >= 64 {
			ph = order.Uint64(h[32:])
			sh = order.Uint64(h[40:])
			eh = order.Uint16(h[52:])
			pent = order.Uint16(h[54:])
			programCount = order.Uint16(h[56:])
			sent = order.Uint16(h[58:])
			sn = order.Uint16(h[60:])
		} else {
			ph = uint64(order.Uint32(h[28:]))
			sh = uint64(order.Uint32(h[32:]))
			eh = order.Uint16(h[40:])
			pent = order.Uint16(h[42:])
			programCount = order.Uint16(h[44:])
			sent = order.Uint16(h[46:])
			sn = order.Uint16(h[48:])
		}
		if err := add(0, uint64(eh), "elf-header"); err != nil {
			return err
		}
		if err := add(ph, uint64(pent)*uint64(programCount), "elf-program-headers"); err != nil {
			return err
		}
		return add(sh, uint64(sent)*uint64(sn), "elf-section-headers")
	}
	if n >= 64 && string(h[:2]) == "MZ" {
		off := uint64(binary.LittleEndian.Uint32(h[60:]))
		if off > size || 88 > size-off {
			return errors.New("PE header outside file")
		}
		var pe [88]byte
		if _, err := r.ReadAt(pe[:], int64(off)); err != nil {
			return err
		}
		if string(pe[:4]) != "PE\x00\x00" {
			return errors.New("invalid PE signature")
		}
		return add(0, uint64(binary.LittleEndian.Uint32(pe[84:88])), "pe-headers")
	}
	if n >= 28 {
		var order binary.ByteOrder = binary.LittleEndian
		magic := order.Uint32(h[:])
		if magic != 0xfeedface && magic != 0xfeedfacf {
			order = binary.BigEndian
			magic = order.Uint32(h[:])
		}
		if magic == 0xfeedface || magic == 0xfeedfacf {
			header := uint64(28)
			if magic == 0xfeedfacf {
				header = 32
			}
			return add(0, header+uint64(order.Uint32(h[20:24])), "macho-load-commands")
		}
	}
	return nil
}
