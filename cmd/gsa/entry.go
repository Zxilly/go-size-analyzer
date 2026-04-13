//go:build !wasm

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/pkg/browser"
	"golang.org/x/exp/mmap"
	"golang.org/x/sync/errgroup"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/diff"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/tui"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

type outputSpec struct {
	format string
	writer io.Writer
	closer io.Closer
}

func inferFormatFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt":
		return printer.FormatText
	case ".json":
		return printer.FormatJSON
	case ".html", ".htm":
		return printer.FormatHTML
	case ".svg":
		return printer.FormatSVG
	}
	return ""
}

// resolveSingleOutput chooses the format for a single bare -o path: explicit
// -f wins; otherwise infer from extension; otherwise text.
func resolveSingleOutput(path string) (outputSpec, error) {
	var format string
	if Options.Format != nil {
		format = *Options.Format
	} else if inferred := inferFormatFromPath(path); inferred != "" {
		format = inferred
	} else {
		format = printer.FormatText
	}
	w, c, err := openPath(path)
	if err != nil {
		return outputSpec{}, err
	}
	return outputSpec{format: format, writer: w, closer: c}, nil
}

func parseOutputs() ([]outputSpec, error) {
	raws := Options.Output

	if len(raws) == 0 {
		format := printer.FormatText
		if Options.Format != nil {
			format = *Options.Format
		}
		return []outputSpec{{format: format, writer: utils.SyncStdout}}, nil
	}

	hasPair, hasBare := false, false
	for _, v := range raws {
		if strings.Contains(v, "=") {
			hasPair = true
		} else {
			hasBare = true
		}
	}
	if hasPair && hasBare {
		return nil, errors.New("-o values must be either all FORMAT=PATH or a single bare path, not mixed")
	}

	if hasBare {
		if len(raws) > 1 {
			return nil, errors.New("-o may only be given once in single-output mode; use FORMAT=PATH to emit multiple formats")
		}
		spec, err := resolveSingleOutput(raws[0])
		if err != nil {
			return nil, err
		}
		return []outputSpec{spec}, nil
	}

	if Options.Format != nil {
		return nil, errors.New("-f cannot be combined with multi-format FORMAT=PATH -o values; the format is carried by each -o")
	}
	if Options.Web || Options.Tui || Options.DiffTarget != "" {
		return nil, errors.New("multi-format -o is not supported with --web, --tui, or diff mode")
	}

	specs := make([]outputSpec, 0, len(raws))
	seenFormat := make(map[string]struct{}, len(raws))
	stdoutUsed := false
	for _, raw := range raws {
		format, path, _ := strings.Cut(raw, "=")
		if !printer.IsSupportedFormat(format) {
			return nil, fmt.Errorf("invalid format %q in -o %q (want %s)", format, raw, strings.Join(printer.SupportedFormats, "|"))
		}
		if _, dup := seenFormat[format]; dup {
			return nil, fmt.Errorf("format %q specified more than once", format)
		}
		if path == "" {
			return nil, fmt.Errorf("empty path in -o %q", raw)
		}
		seenFormat[format] = struct{}{}

		if path == "-" {
			if stdoutUsed {
				return nil, errors.New("at most one output may be written to stdout")
			}
			stdoutUsed = true
			specs = append(specs, outputSpec{format: format, writer: utils.SyncStdout})
			continue
		}

		w, c, err := openPath(path)
		if err != nil {
			closeAll(specs)
			return nil, err
		}
		specs = append(specs, outputSpec{format: format, writer: w, closer: c})
	}
	return specs, nil
}

func openPath(path string) (io.Writer, io.Closer, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func closeAll(specs []outputSpec) {
	for _, s := range specs {
		if s.closer != nil {
			_ = s.closer.Close()
		}
	}
}

func renderOne(spec outputSpec, r *result.Result, common printer.CommonOption) error {
	switch spec.format {
	case printer.FormatText:
		return printer.Text(r, spec.writer, &common)
	case printer.FormatJSON:
		return printer.JSON(r, spec.writer, &printer.JSONOption{
			Indent:     Options.Indent,
			HideDetail: Options.Compact,
		})
	case printer.FormatHTML:
		return printer.HTML(r, spec.writer)
	case printer.FormatSVG:
		return printer.Svg(r, spec.writer, &printer.SvgOption{
			CommonOption: common,
			Width:        Options.Width,
			Height:       Options.Height,
			MarginBox:    Options.MarginBox,
			PaddingBox:   Options.PaddingBox,
			PaddingRoot:  Options.PaddingRoot,
		})
	default:
		return fmt.Errorf("invalid format: %s", spec.format)
	}
}

func entry() error {
	options := internal.Options{
		SkipSymbol: Options.NoSymbol,
		SkipDisasm: Options.NoDisasm,
		SkipDwarf:  Options.NoDwarf,
		Imports:    Options.Imports,
	}

	if Options.DiffTarget != "" {
		for _, o := range Options.Output {
			if strings.Contains(o, "=") {
				return errors.New("diff mode does not accept FORMAT=PATH -o values")
			}
		}
		if len(Options.Output) > 1 {
			return errors.New("diff mode accepts at most one -o path")
		}
		writer := io.Writer(utils.SyncStdout)
		format := printer.FormatText
		if Options.Format != nil {
			format = *Options.Format
		}
		if len(Options.Output) == 1 {
			spec, err := resolveSingleOutput(Options.Output[0])
			if err != nil {
				return err
			}
			defer spec.closer.Close()
			writer = spec.writer
			format = spec.format
		}
		return diff.Diff(writer, diff.Options{
			Options:   options,
			OldTarget: Options.Binary,
			NewTarget: Options.DiffTarget,
			Format:    format,
			Indent:    Options.Indent,
		})
	}

	specs, err := parseOutputs()
	if err != nil {
		return err
	}
	defer closeAll(specs)

	var webBuf *bytes.Buffer
	if Options.Web {
		if len(specs) != 1 {
			return errors.New("--web is not compatible with multi-format -o")
		}
		webBuf = new(bytes.Buffer)
		specs = []outputSpec{{format: printer.FormatHTML, writer: webBuf}}
	}

	reader, err := mmap.Open(Options.Binary)
	if err != nil {
		return err
	}

	r, err := internal.Analyze(Options.Binary,
		reader,
		uint64(reader.Len()),
		options)
	if err != nil {
		return err
	}

	if err := reader.Close(); err != nil {
		return err
	}

	if Options.Tui {
		w, h, err := term.GetSize(os.Stdout.Fd())
		if err != nil {
			return fmt.Errorf("failed to get terminal size: %w", err)
		}
		return tui.RunTUI(r, w, h)
	}

	common := printer.CommonOption{
		HideSections: Options.HideSections,
		HideMain:     Options.HideMain,
		HideStd:      Options.HideStd,
	}

	if len(specs) == 1 {
		if err := renderOne(specs[0], r, common); err != nil {
			return err
		}
	} else {
		var eg errgroup.Group
		for _, spec := range specs {
			eg.Go(func() error { return renderOne(spec, r, common) })
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}

	slog.Info("Printing done")

	if Options.Web {
		slog.Debug("Starting web server")

		webui.HostServer(webBuf.Bytes(), Options.Listen)

		url := utils.GetURLFromListen(Options.Listen)

		slog.Info("Server started at " + url)

		if Options.Open {
			err = browser.OpenURL(url)
			if err != nil {
				slog.Warn(fmt.Sprintf("Failed to open: %v", err))
			}
		}

		utils.WaitSignal()
	}

	slog.Info("Ready to exit")

	return nil
}
