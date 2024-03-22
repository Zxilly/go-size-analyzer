package printer

import (
	"cmp"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
	"slices"
)

func percentString(f float64) string {
	return fmt.Sprintf("%.2f%%", f)
}

func PrintResult(r *internal.Result) {
	t := table.NewWriter()
	t.SetOutputMirror(utils.Stdout)

	knownSize := uint64(0)

	t.SetTitle("%#v", r.Name)
	t.AppendHeader(table.Row{"Percent", "Package", "Size", "Type"})

	pkgs := maps.Values(r.Packages)
	slices.SortFunc(pkgs, func(a, b *internal.Package) int {
		return -cmp.Compare(a.Size, b.Size)
	})
	for _, p := range pkgs {
		knownSize += p.Size
		t.AppendRow(table.Row{percentString(float64(p.Size) / float64(r.Size) * 100), p.Name, humanize.Bytes(p.Size), p.Type})
	}

	t.Render()
	t = table.NewWriter()
	t.SetOutputMirror(utils.Stdout)

	t.AppendHeader(table.Row{"Percent", "Unknown section", "Size"})
	sections := lo.Filter(r.Sections, func(s *internal.Section, _ int) bool {
		return s.Size > s.KnownSize && s.Size != s.KnownSize && !s.OnlyInMemory
	})
	slices.SortFunc(sections, func(a, b *internal.Section) int {
		return -cmp.Compare(a.Size-a.KnownSize, b.Size-b.KnownSize)
	})
	for _, s := range sections {
		unknownSize := s.Size - s.KnownSize
		knownSize += unknownSize
		t.AppendRow(table.Row{percentString(float64(unknownSize) / float64(r.Size) * 100), s.Name, humanize.Bytes(unknownSize)})
	}

	t.AppendFooter(table.Row{percentString(float64(knownSize) / float64(r.Size) * 100), "Known", humanize.Bytes(knownSize)})
	t.AppendFooter(table.Row{"100%", "Total", humanize.Bytes(r.Size)})
	t.Render()
}
