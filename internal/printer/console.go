//go:build !js && !wasm

package printer

import (
	"cmp"
	"io"
	"log/slog"
	"maps"
	"slices"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type CommonOption struct {
	HideSections bool
	HideMain     bool
	HideStd      bool
}

func Text(r *result.Result, writer io.Writer, options *CommonOption) error {
	slog.Info("Printing text report")

	t := table.NewWriter()
	t.SetStyle(utils.GetTableStyle())

	allKnownSize := uint64(0)

	t.SetTitle("%s", r.Name)
	t.AppendHeader(table.Row{"Percent", "Name", "Size", "Type"})

	type sizeEntry struct {
		name    string
		size    uint64
		typ     string
		percent string
	}

	entries := make([]sizeEntry, 0)

	pkgs := utils.Collect(maps.Values(r.Packages))
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
			percent: utils.PercentString(float64(p.Size) / float64(r.Size)),
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
				percent: utils.PercentString(float64(unknownSize) / float64(r.Size)),
			})
		}
	}

	allKnownSize = min(allKnownSize, r.Size) // since we can have overlap

	slices.SortFunc(entries, func(a, b sizeEntry) int {
		return -cmp.Compare(a.size, b.size)
	})

	for _, e := range entries {
		t.AppendRow(table.Row{e.percent, e.name, humanize.Bytes(e.size), e.typ})
	}

	t.AppendFooter(table.Row{utils.PercentString(float64(allKnownSize) / float64(r.Size)), "Known", humanize.Bytes(allKnownSize)})
	t.AppendFooter(table.Row{"100%", "Total", humanize.Bytes(r.Size)})

	data := []byte(t.Render() + "\n")

	slog.Info("Report rendered")

	_, err := writer.Write(data)

	slog.Info("Report written")

	return err
}
