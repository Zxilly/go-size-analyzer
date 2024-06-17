//go:build (js && wasm) || test_js_marshaler

package entity

import (
	"github.com/samber/lo"
)

func (m PackageMap) MarshalJavaScript() any {
	ret := map[string]any{}

	for k, v := range m {
		ret[k] = v.MarshalJavaScript()
	}

	return ret
}

func (p *Package) MarshalJavaScript() any {
	var symbols, files []any
	symbols = lo.Map(p.Symbols, func(s *Symbol, _ int) any { return s.MarshalJavaScript() })
	files = lo.Map(p.Files, func(f *File, _ int) any { return f.MarshalJavaScript() })
	subs := p.SubPackages.MarshalJavaScript()

	return map[string]any{
		"name":        p.Name,
		"type":        p.Type,
		"size":        p.Size,
		"symbols":     symbols,
		"subPackages": subs,
		"files":       files,
	}
}
