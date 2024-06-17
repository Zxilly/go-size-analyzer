//go:build (js && wasm) || test_js_marshaler

package entity

func (f *File) MarshalJavaScript() any {
	return map[string]any{
		"file_path": f.FilePath,
		"size":      f.FullSize(),
		"pcln_size": f.PclnSize(),
	}
}
