//go:build js && wasm

package entity

import (
	"syscall/js"
)

func (s *Symbol) MarshalJavaScript() js.Value {
	return js.ValueOf(map[string]any{
		"name": s.Name,
		"addr": s.Addr,
		"size": s.Size,
		"type": s.Type,
	})
}
