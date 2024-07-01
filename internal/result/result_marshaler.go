//go:build js && wasm

package result

func (r *Result) MarshalJavaScript() any {
	var sections []any
	for _, s := range r.Sections {
		sections = append(sections, s.MarshalJavaScript())
	}
	var analyzers []any
	for _, a := range r.Analyzers {
		analyzers = append(analyzers, a)
	}

	packages := r.Packages.MarshalJavaScript()

	return map[string]any{
		"name":      r.Name,
		"size":      r.Size,
		"packages":  packages,
		"sections":  sections,
		"analyzers": analyzers,
	}
}
