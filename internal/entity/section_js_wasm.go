//go:build js && wasm

package entity

import (
	"syscall/js"
)

func (s Section) MarshalJavaScript() js.Value {
	return js.ValueOf(map[string]any{
		"name":           s.Name,
		"size":           s.Size,
		"file_size":      s.FileSize,
		"known_size":     s.KnownSize,
		"offset":         s.Offset,
		"end":            s.End,
		"addr":           s.Addr,
		"addr_end":       s.AddrEnd,
		"only_in_memory": s.OnlyInMemory,
		"debug":          s.Debug,
	})
}
