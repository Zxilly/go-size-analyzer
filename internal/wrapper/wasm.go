package wrapper

import (
	"debug/dwarf"
	"errors"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type WasmWrapper struct {
	module *gore.WasmModule
}

var _ RawFileWrapper = (*WasmWrapper)(nil)

const funcValueOffset = 0x1000

func (w *WasmWrapper) GetFunctionSize(idx uint64) uint64 {
	// get PC_F from PC
	idx = idx >> 16
	idx = idx - funcValueOffset

	return uint64(len(w.module.CodeSection[idx].Body))
}

func (w *WasmWrapper) Text() (textStart uint64, text []byte, err error) {
	err = errors.New("text section not supported")
	return
}

func (w *WasmWrapper) GoArch() string {
	return "wasm"
}

func (w *WasmWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	return nil, errors.New("read addr not supported")
}

func (w *WasmWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType), goSCb func(addr uint64, size uint64)) error {
	return errors.New("load symbols not supported")
}

func (w *WasmWrapper) LoadSections() *entity.Store {
	return nil
}

func (w *WasmWrapper) DWARF() (*dwarf.Data, error) {
	return nil, errors.New("dwarf section not supported")
}

var _ RawFileWrapper = (*WasmWrapper)(nil)
