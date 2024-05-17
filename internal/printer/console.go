package printer

import (
	"cmp"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func percentString(f float64) string {
	return fmt.Sprintf("%.2f%%", f)
}

type CommonOption struct {
	Writer       io.Writer
	HideSections bool
	HideMain     bool
	HideStd      bool
}

func Text(r *result.Result, options *CommonOption) error {
	t := table.NewWriter()

	allKnownSize := uint64(0)

	t.SetTitle("%s", filepath.Base(r.Name))
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
		if options.HideMain && p.Type == entity.PackageTypeMain {
			continue
		}
		if options.HideStd && p.Type == entity.PackageTypeStd {
			continue
		}

		allKnownSize += p.Size
		entries = append(entries, sizeEntry{
			name:    p.Name,
			size:    p.Size,
			typ:     p.Type,
			percent: percentString(float64(p.Size) / float64(r.Size) * 100),
		})
	}

	if !options.HideSections {
		sections := lo.Filter(r.Sections, func(s *entity.Section, _ int) bool {
			return s.Size > s.KnownSize && s.Size != s.KnownSize && !s.OnlyInMemory
		})
		for _, s := range sections {
			unknownSize := s.FileSize - s.KnownSize
			allKnownSize += unknownSize
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

	t.AppendFooter(table.Row{percentString(float64(allKnownSize) / float64(r.Size) * 100), "Known", humanize.Bytes(allKnownSize)})
	t.AppendFooter(table.Row{"100%", "Total", humanize.Bytes(r.Size)})

	_, err := options.Writer.Write([]byte(t.Render()))
	return err
}
