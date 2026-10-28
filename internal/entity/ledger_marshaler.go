//go:build js && wasm

package entity

func (c *FileCoverage) MarshalJavaScript() any {
	sources := map[string]any{}
	for source, size := range c.BySource {
		sources[source] = size
	}
	result := map[string]any{"attributed": c.Attributed, "recognized": c.Recognized, "unclassified": c.Unclassified, "shared": c.Shared, "by_source": sources}
	if c.Version != 0 {
		result["version"] = c.Version
	}
	if len(c.Notes) > 0 {
		notes := make([]any, len(c.Notes))
		for i, note := range c.Notes {
			notes[i] = note
		}
		result["notes"] = notes
	}
	if len(c.Regions) > 0 {
		regions := make([]any, 0, len(c.Regions))
		for _, r := range c.Regions {
			item := map[string]any{"offset": r.Offset, "size": r.Size, "class": r.Class}
			if len(r.Owners) > 0 {
				owners := make([]any, len(r.Owners))
				for i, v := range r.Owners {
					owners[i] = v
				}
				item["owners"] = owners
			}
			if len(r.Sources) > 0 {
				sources := make([]any, len(r.Sources))
				for i, v := range r.Sources {
					sources[i] = v
				}
				item["sources"] = sources
			}
			for name, values := range map[string][]string{"definition_owners": r.DefinitionOwners, "referrers": r.Referrers} {
				if len(values) > 0 {
					list := make([]any, len(values))
					for i, value := range values {
						list[i] = value
					}
					item[name] = list
				}
			}
			if len(r.Evidence) > 0 {
				evidence := make([]any, 0, len(r.Evidence))
				for _, e := range r.Evidence {
					value := map[string]any{"owner": e.Owner, "source": e.Source, "role": e.Role}
					if e.Boundary != "" {
						value["boundary"] = e.Boundary
					}
					evidence = append(evidence, value)
				}
				item["evidence"] = evidence
			}
			regions = append(regions, item)
		}
		result["regions"] = regions
	}
	return result
}
