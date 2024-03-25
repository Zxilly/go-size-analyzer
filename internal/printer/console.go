package printer

import (
	"cmp"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
	"slices"
)

func percentString(f float64) string {
	return fmt.Sprintf("%.2f%%", f)
}

func Text(r *internal.Result, options *TextOption) string {
	t := table.NewWriter()

	knownSize := uint64(0)

	t.SetTitle("%s", r.Name)
	t.AppendHeader(table.Row{"Percent", "Name", "Size", "Type"})

	type sizeEntry struct {
		name    string
		size    uint64
		typ     string
		percent string
	}

	entries := make([]sizeEntry, 0)

	pkgs := maps.Values(r.Packages)
	for _, p := range pkgs {
		if options.HideMain && p.Type == internal.PackageTypeMain {
			continue
		}
		if options.HideStd && p.Type == internal.PackageTypeStd {
			continue
		}

		knownSize += p.Size
		entries = append(entries, sizeEntry{
			name:    p.Name,
			size:    p.Size,
			typ:     p.Type,
			percent: percentString(float64(p.Size) / float64(r.Size) * 100),
		})
	}

	if !options.HideSections {
		sections := lo.Filter(r.Sections, func(s *internal.Section, _ int) bool {
			return s.Size > s.KnownSize && s.Size != s.KnownSize && !s.OnlyInMemory
		})
		for _, s := range sections {
			unknownSize := s.Size - s.KnownSize
			knownSize += unknownSize
			entries = append(entries, sizeEntry{
				name:    s.Name,
				size:    unknownSize,
				typ:     "section",
				percent: percentString(float64(unknownSize) / float64(r.Size) * 100),
			})
		}
	}

	slices.SortFunc(entries, func(a, b sizeEntry) int {
		return -cmp.Compare(a.size, b.size)
	})

	for _, e := range entries {
		t.AppendRow(table.Row{e.percent, e.name, humanize.Bytes(e.size), e.typ})
	}

	t.AppendFooter(table.Row{percentString(float64(knownSize) / float64(r.Size) * 100), "Known", humanize.Bytes(knownSize)})
	t.AppendFooter(table.Row{"100%", "Total", humanize.Bytes(r.Size)})

	return t.Render()
}
