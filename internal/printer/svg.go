package printer

import (
	"encoding/csv"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/nikolaydubina/treemap"
	"github.com/nikolaydubina/treemap/parser"
	"github.com/nikolaydubina/treemap/render"
	"image/color"
	"strconv"
	"strings"
)

type SvgOption struct {
	CommonOption
	Width       int
	Height      int
	MarginBox   int
	PaddingBox  int
	PaddingRoot int
}

func Svg(r *result.Result, options *SvgOption) []byte {
	s := new(strings.Builder)
	c := csv.NewWriter(s)

	// write file
	_ = c.Write([]string{r.Name, strconv.Itoa(int(r.Size)), "0"})

	merge := func(s string) string {
		return r.Name + "/" + s
	}

	// write sections
	if !options.HideSections {
		for _, sec := range r.Sections {
			_ = c.Write([]string{merge(sec.Name), strconv.Itoa(int(sec.Size)), "0"})
		}
	}

	// write packages
	var writePackage func(p *entity.Package)
	writePackage = func(p *entity.Package) {
		if !((options.HideMain && p.Type == entity.PackageTypeMain) ||
			(options.HideStd && p.Type == entity.PackageTypeStd)) {
			_ = c.Write([]string{merge(p.Name), strconv.Itoa(int(p.Size)), "0"})
		}
		for _, sub := range p.SubPackages {
			writePackage(sub)
		}
	}

	for _, p := range r.Packages {
		writePackage(p)
	}

	c.Flush()

	p := parser.CSVTreeParser{}
	tree, err := p.ParseString(s.String())
	if err != nil {
		panic(err)
	}

	treemap.SetNamesFromPaths(tree)
	treemap.CollapseLongPaths(tree)

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)

	tree.NormalizeHeat()

	colorer := render.NoneColorer{}
	borderColor := color.RGBA{128, 128, 128, 255}

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

	return renderer.Render(spec, float64(options.Width), float64(options.Height))
}
