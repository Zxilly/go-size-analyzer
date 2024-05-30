//go:build js && wasm

package entity

import (
	"syscall/js"
)

func (f *File) MarshalJavaScript() js.Value {
	return js.ValueOf(map[string]any{
		"file_path": f.FilePath,
		"size":      f.FullSize(),
		"pcln_size": f.PclnSize(),
	})
}
