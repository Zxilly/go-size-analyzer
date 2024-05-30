//go:build js && wasm

package entity

import (
	"syscall/js"

	"github.com/samber/lo"
)

func (m PackageMap) MarshalJavaScript() js.Value {
	ret := map[string]any{}

	for k, v := range m {
		ret[k] = v.MarshalJavaScript()
	}

	return js.ValueOf(ret)
}

func (p *Package) MarshalJavaScript() js.Value {
	var symbols, files []any
	symbols = lo.Map(p.Symbols, func(s *Symbol, _ int) any { return s.MarshalJavaScript() })
	files = lo.Map(p.Files, func(f *File, _ int) any { return f.MarshalJavaScript() })

	return js.ValueOf(map[string]any{
		"name":        p.Name,
		"type":        p.Type,
		"size":        p.Size,
		"symbols":     symbols,
		"subPackages": p.SubPackages.MarshalJavaScript(),
		"files":       files,
	})
}
