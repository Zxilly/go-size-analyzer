package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/pkg/browser"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/tui"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

func entry() {
	utils.ApplyMemoryLimit()

	if Options.Verbose {
		utils.InitLogger(slog.LevelDebug)
	} else {
		utils.InitLogger(slog.LevelWarn)
	}

	result, err := internal.Analyze(Options.Binary, internal.Options{
		SkipSymbol: Options.NoSymbol,
		SkipDisasm: Options.NoDisasm,
	})
	if err != nil {
		utils.FatalError(err)
	}

	if Options.Tui {
		tui.RunTUI(result)
		return
	}

	if Options.Web {
		Options.Format = "html"
	}

	var writer io.Writer

	if Options.Output != "" {
		writer, err = os.OpenFile(Options.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			utils.FatalError(err)
		}
	} else {
		writer = os.Stdout
		if Options.Web {
			writer = new(bytes.Buffer)
		}
	}

	common := printer.CommonOption{
		HideSections: Options.HideSections,
		HideMain:     Options.HideMain,
		HideStd:      Options.HideStd,
		Writer:       writer,
	}

	switch Options.Format {
	case "text":
		err = printer.Text(result, &common)
	case "json":
		err = printer.JSON(result, &printer.JSONOption{
			Indent:     Options.Indent,
			HideDetail: Options.Compact,
			Writer:     writer,
		})
	case "html":
		err = printer.HTML(result, writer)
	case "svg":
		err = printer.Svg(result, &printer.SvgOption{
			CommonOption: common,
			Width:        Options.Width,
			Height:       Options.Height,
			MarginBox:    Options.MarginBox,
			PaddingBox:   Options.PaddingBox,
			PaddingRoot:  Options.PaddingRoot,
		})
	default:
		utils.FatalError(fmt.Errorf("invalid format: %s", Options.Format))
	}

	if err != nil {
		utils.FatalError(err)
	}

	if Options.Web {
		b, ok := writer.(*bytes.Buffer)
		if !ok {
			panic("writer is not bytes.Buffer")
		}

		webui.HostServer(b.Bytes(), Options.Listen)

		url := utils.GetURLFromListen(Options.Listen)

		fmt.Println("Server started at", url)

		if Options.Open {
			err = browser.OpenURL(url)
			if err != nil {
				slog.Warn(fmt.Sprintf("Failed to open: %v", err))
			}
		}

		utils.WaitSignal()
		return
	}
}
