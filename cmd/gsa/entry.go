//go:build !wasm

package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/pkg/browser"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/diff"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/tui"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

func entry() error {
	options := internal.Options{
		SkipSymbol: Options.NoSymbol,
		SkipDisasm: Options.NoDisasm,
		SkipDwarf:  Options.NoDwarf,
	}

	var writer io.Writer
	var err error

	if Options.Output != "" {
		writer, err = os.OpenFile(Options.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
	} else {
		writer = utils.Stdout
		if Options.Web {
			writer = new(bytes.Buffer)
		}
	}

	if Options.DiffTarget != "" {
		return diff.Diff(writer, diff.DOptions{
			Options:   options,
			OldTarget: Options.Binary,
			NewTarget: Options.DiffTarget,
			Format:    Options.Format,
			Indent:    Options.Indent,
		})
	}

	reader, err := mmap.Open(Options.Binary)
	if err != nil {
		return err
	}

	result, err := internal.Analyze(Options.Binary,
		reader,
		uint64(reader.Len()),
		options)
	if err != nil {
		return err
	}

	err = reader.Close()
	if err != nil {
		return err
	}

	if Options.Tui {
		w, h, err := term.GetSize(os.Stdout.Fd())
		if err != nil {
			return fmt.Errorf("failed to get terminal size: %w", err)
		}

		return tui.RunTUI(result, w, h)
	}

	if Options.Web {
		Options.Format = "html"
	}

	common := printer.CommonOption{
		HideSections: Options.HideSections,
		HideMain:     Options.HideMain,
		HideStd:      Options.HideStd,
	}

	switch Options.Format {
	case "text":
		err = printer.Text(result, writer, &common)
	case "json":
		err = printer.JSON(result, writer, &printer.JSONOption{
			Indent:     Options.Indent,
			HideDetail: Options.Compact,
		})
	case "html":
		err = printer.HTML(result, writer)
	case "svg":
		err = printer.Svg(result, writer, &printer.SvgOption{
			CommonOption: common,
			Width:        Options.Width,
			Height:       Options.Height,
			MarginBox:    Options.MarginBox,
			PaddingBox:   Options.PaddingBox,
			PaddingRoot:  Options.PaddingRoot,
		})
	default:
		return fmt.Errorf("invalid format: %s", Options.Format)
	}

	slog.Info("Printing done")

	if err != nil {
		return err
	}

	if Options.Web {
		slog.Debug("Starting web server")

		b, ok := writer.(*bytes.Buffer)
		if !ok {
			panic("writer is not bytes.Buffer")
		}

		webui.HostServer(b.Bytes(), Options.Listen)

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
