package result_test

import (
	"encoding/gob"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func init() {
	gob.Register(entity.GoPclntabMeta{})
	gob.Register(entity.SymbolMeta{})
	gob.Register(entity.DisasmMeta{})
	gob.Register(entity.DwarfMeta{})
}
