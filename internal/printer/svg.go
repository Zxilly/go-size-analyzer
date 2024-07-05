//go:build !js && !wasm

package printer

import (
	"image/color"
	"io"

	"github.com/nikolaydubina/treemap"
	"github.com/nikolaydubina/treemap/render"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

type SvgOption struct {
	CommonOption
	Width       int
	Height      int
	MarginBox   int
	PaddingBox  int
	PaddingRoot int
}

func Svg(r *result.Result, writer io.Writer, options *SvgOption) error {
	baseName := r.Name

	tree := &treemap.Tree{
		Nodes: make(map[string]treemap.Node),
		To:    make(map[string][]string),
		Root:  baseName,
	}

	insert := func(path string, size float64) {
		tree.Nodes[path] = treemap.Node{
			Path: path,
			Size: size,
		}
	}

	relation := func(parent, child string) {
		tree.To[parent] = append(tree.To[parent], child)
	}

	merge := func(s string) string {
		return baseName + "/" + s
	}

	// write file
	insert(baseName, float64(r.Size))

	// write sections
	if !options.HideSections {
		for _, sec := range r.Sections {
			insert(merge(sec.Name), float64(sec.FileSize-sec.KnownSize))
			relation(baseName, merge(sec.Name))
		}
	}

	// write packages
	var writePackage func(p *entity.Package)
	writePackage = func(p *entity.Package) {
		if !((options.HideMain && p.Type == entity.PackageTypeMain) ||
			(options.HideStd && p.Type == entity.PackageTypeStd)) {
			insert(merge(p.Name), float64(p.Size))
		}
		for _, sub := range p.SubPackages {
			relation(merge(p.Name), merge(sub.Name))
			writePackage(sub)
		}
	}

	for _, p := range r.Packages {
		writePackage(p)
		relation(baseName, merge(p.Name))
	}

	treemap.SetNamesFromPaths(tree)
	treemap.CollapseLongPaths(tree)

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)

	tree.NormalizeHeat()

	colorer := render.NoneColorer{}
	borderColor := color.RGBA{R: 128, G: 128, B: 128, A: 255}

	uiBuilder := render.UITreeMapBuilder{
		Colorer:     colorer,
		BorderColor: borderColor,
	}
	spec := uiBuilder.NewUITreeMap(*tree,
		float64(options.Width),
		float64(options.Height),
		float64(options.MarginBox),
		float64(options.PaddingBox),
		float64(options.PaddingRoot))
	renderer := render.SVGRenderer{}

	data := renderer.Render(spec, float64(options.Width), float64(options.Height))
	_, err := writer.Write(data)
	return err
}
