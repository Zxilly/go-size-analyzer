//go:build (js && wasm) || test_js_marshaler

package entity

func (s *Symbol) MarshalJavaScript() any {
	return map[string]any{
		"name": s.Name,
		"addr": s.Addr,
		"size": s.Size,
		"type": s.Type,
	}
}
