package wrapper

import (
	"debug/dwarf"
	"errors"
	"log/slog"
	"strings"

	"github.com/ZxillyFork/wazero/notinternal/wasm"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type WasmWrapper struct {
	module *wasm.Module
}

var _ RawFileWrapper = (*WasmWrapper)(nil)

const funcValueOffset = 0x1000

func (w *WasmWrapper) GetFunctionSize(idx uint64) uint64 {
	// get PC_F from PC
	idx = idx >> 16
	idx = idx - funcValueOffset

	return uint64(len(w.module.CodeSection[idx].Body))
}

func (*WasmWrapper) Text() (textStart uint64, text []byte, err error) {
	return textStart, nil, errors.New("text section not supported")
}

func (*WasmWrapper) GoArch() string {
	return "wasm"
}

func (*WasmWrapper) ReadAddr(uint64, uint64) ([]byte, error) {
	return nil, errors.New("read addr not supported")
}

func (*WasmWrapper) LoadSymbols(func(name string, addr uint64, size uint64, typ entity.AddrType), func(addr uint64, size uint64)) error {
	return errors.New("load symbols not supported")
}

func (*WasmWrapper) LoadSections() *entity.Store {
	return nil
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

var _ RawFileWrapper = (*WasmWrapper)(nil)
