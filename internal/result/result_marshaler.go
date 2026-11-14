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

	value := map[string]any{
		"name":      r.Name,
		"size":      r.Size,
		"packages":  packages,
		"sections":  sections,
		"analyzers": analyzers,
	}
	if r.Coverage != nil {
		// The UI consumes coverage summaries, not the complete physical ledger.
		// Keep the original available to full JSON and other consumers.
		coverage := *r.Coverage
		coverage.Regions = nil
		value["coverage"] = coverage.MarshalJavaScript()
	}
	if r.BuildInfo != nil {
		value["build_info"] = buildInfoJavaScript(r.BuildInfo)
	}
	return value
}
