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
	path   string
	file   *os.File
	temp   string
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
	if inferred := inferFormatFromPath(path); Options.Format != nil && inferred != "" && inferred != format {
		return outputSpec{}, fmt.Errorf("format %s conflicts with output extension %s", format, filepath.Ext(path))
	}
	return outputSpec{format: format, path: path}, nil
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

		specs = append(specs, outputSpec{format: format, path: path})
	}
	return specs, nil
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
	parseCommandLine()
	return run()
}

func run() error {
	options := internal.Options{
		SkipSymbol: Options.NoSymbol,
		SkipDisasm: Options.NoDisasm,
		SkipDwarf:  Options.NoDwarf,
		Imports:    Options.Imports,
	}

	specs, err := parseOutputs()
	if err != nil {
		return err
	}

	var webBuf *bytes.Buffer
	if Options.Web {
		if len(Options.Output) > 0 {
			return errors.New("--web does not write report files")
		}
		if len(specs) != 1 {
			return errors.New("--web is not compatible with multi-format -o")
		}
		webBuf = new(bytes.Buffer)
		specs = []outputSpec{{format: printer.FormatHTML, writer: webBuf}}
	}

	if Options.Tui && len(Options.Output) > 0 {
		return errors.New("--tui does not write report files")
	}
	if Options.DiffTarget != "" && specs[0].format != printer.FormatJSON && specs[0].format != printer.FormatText {
		return errors.New("diff mode only supports text and json output")
	}
	if err := prepareOutputs(specs); err != nil {
		return err
	}
	defer closeAll(specs)
	if Options.DiffTarget != "" {
		if err := diff.Diff(specs[0].writer, diff.Options{
			Options: options, OldTarget: Options.Binary, NewTarget: Options.DiffTarget,
			Format: specs[0].format, Indent: Options.Indent,
		}); err != nil {
			return err
		}
		return commitOutputs(specs)
	}
	reader, err := utils.OpenBinary(Options.Binary)
	if err != nil {
		return fmt.Errorf("open binary %s: %w", Options.Binary, err)
	}

	r, err := internal.Analyze(Options.Binary,
		reader,
		uint64(reader.Len()),
		options)
	if err != nil {
		_ = reader.Close()
		return fmt.Errorf("analyze %s: %w", Options.Binary, err)
	}

	if err := reader.Close(); err != nil {
		return fmt.Errorf("close %s: %w", Options.Binary, err)
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

	if err := commitOutputs(specs); err != nil {
		return err
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
