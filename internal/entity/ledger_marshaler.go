//go:build js && wasm

package entity

func (c *FileCoverage) MarshalJavaScript() any {
	sources := map[string]any{}
	for source, size := range c.BySource {
		sources[source] = size
	}
	result := map[string]any{"attributed": c.Attributed, "recognized": c.Recognized, "unclassified": c.Unclassified, "shared": c.Shared, "by_source": sources}
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
			regions = append(regions, item)
		}
		result["regions"] = regions
	}
	return result
}
