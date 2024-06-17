//go:build (js && wasm) || test_js_marshaler

package result

func (r *Result) MarshalJavaScript() any {
	var sections []any
	for _, s := range r.Sections {
		sections = append(sections, s.MarshalJavaScript())
	}

	packages := r.Packages.MarshalJavaScript()

	return map[string]any{
		"name":     r.Name,
		"size":     r.Size,
		"packages": packages,
		"sections": sections,
	}
}
