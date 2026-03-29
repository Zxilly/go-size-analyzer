package wrapper

import (
	"bytes"
	"debug/dwarf"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/ZxillyFork/wazero/notinternal/wasm"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type WasmWrapper struct {
	module *wasm.Module
	memory []byte
}

var _ RawFileWrapper = (*WasmWrapper)(nil)

const funcValueOffset = 0x1000

func (w *WasmWrapper) GetFunctionSize(idx uint64, meq125 bool) uint64 {
	// Go 1.25+ stores PC_F directly in pclntab, older versions store full PC (PC_F << 16)
	if !meq125 {
		idx = idx >> 16
	}
	idx = idx - funcValueOffset

	return uint64(len(w.module.CodeSection[idx].Body))
}

func (*WasmWrapper) Text() (textStart uint64, text []byte, err error) {
	return textStart, nil, errors.New("text section not supported")
}

func (*WasmWrapper) GoArch() string {
	return "wasm"
}

func (w *WasmWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	end := addr + size
	if end > uint64(len(w.memory)) || end < addr {
		return nil, fmt.Errorf("read addr 0x%x size 0x%x out of range (memory size 0x%x)", addr, size, len(w.memory))
	}
	return w.memory[addr:end], nil
}

func (*WasmWrapper) LoadSymbols(func(name string, addr uint64, size uint64, typ entity.AddrType), func(addr uint64, size uint64)) error {
	return errors.New("load symbols not supported")
}

func (w *WasmWrapper) LoadSections() *entity.Store {
	store := entity.NewStore()

	// The linear memory acts as a single data section for type/pclntab analysis.
	// Type descriptors and pclntab metadata use offsets into this memory.
	memSize := uint64(len(w.memory))
	if memSize > 0 {
		store.Sections["memory.data"] = &entity.Section{
			Name:         "memory.data",
			Size:         memSize,
			Addr:         0,
			AddrEnd:      memSize,
			OnlyInMemory: true,
			ContentType:  entity.SectionContentData,
		}
	}

	return store
}

func (*WasmWrapper) DWARF() (*dwarf.Data, error) {
	return nil, errors.New("dwarf section not supported")
}

func (w *WasmWrapper) GetSections(codeSectUsed uint64) []*entity.Section {
	ret := make([]*entity.Section, 0)
	for name, sect := range w.module.Sections {
		knownSize := uint64(0)
		if name == "code" {
			if codeSectUsed <= uint64(sect.Size) {
				knownSize = codeSectUsed
			} else {
				knownSize = uint64(sect.Size)
				slog.Warn("known code size is greater than code section size")
			}
		}

		ret = append(ret, &entity.Section{
			Name:         name,
			Size:         uint64(sect.Size),
			FileSize:     uint64(sect.Size),
			KnownSize:    knownSize,
			Offset:       uint64(sect.Offset),
			End:          uint64(sect.Offset) + uint64(sect.Size),
			Addr:         0,
			AddrEnd:      0,
			OnlyInMemory: true,
			Debug:        strings.HasPrefix(name, "custom_.debug"),
		})
	}
	return ret
}

var (
	infoStart, _ = hex.DecodeString("3077af0c9274080241e1c107e6d618e6")
	infoEnd, _   = hex.DecodeString("f932433186182072008242104116d8f2")
)

func (w *WasmWrapper) GetModInfo() *debug.BuildInfo {
	data := w.memory

	startMarkerLocation := bytes.Index(data, infoStart)
	if startMarkerLocation == -1 {
		return nil
	}

	searchForEndMarkerFrom := startMarkerLocation + len(infoStart)
	if searchForEndMarkerFrom > len(data) {
		return nil
	}

	remainingData := data[searchForEndMarkerFrom:]
	endMarkerRelativeLocation := bytes.Index(remainingData, infoEnd)

	if endMarkerRelativeLocation == -1 {
		return nil
	}

	sliceEndIndex := searchForEndMarkerFrom + endMarkerRelativeLocation + len(infoEnd)

	modinfo := string(data[startMarkerLocation:sliceEndIndex])

	bi, err := debug.ParseBuildInfo(modinfo)
	if err != nil {
		return nil
	}
	return bi
}

var _ RawFileWrapper = (*WasmWrapper)(nil)
