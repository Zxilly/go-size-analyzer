//go:build (js && wasm) || test_js_marshaler

package entity

func (s Section) MarshalJavaScript() any {
	return map[string]any{
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
	}
}
