//go:build js && wasm

package entity

func (f *File) MarshalJavaScript() any {
	functions := make([]any, 0, len(f.Functions))
	for _, fn := range f.Functions {
		functions = append(functions, fn.MarshalJavaScript())
	}
	return map[string]any{
		"file_path": f.FilePath,
		"size":      f.FullSize(),
		"pcln_size": f.PclnSize(),
		"functions": functions,
	}
}
