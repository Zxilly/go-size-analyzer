package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pkg/browser"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/tui"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

func main() {
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

	var b []byte
	common := printer.CommonOption{
		HideSections: Options.HideSections,
		HideMain:     Options.HideMain,
		HideStd:      Options.HideStd,
	}

	switch Options.Format {
	case "text":
		b = []byte(printer.Text(result, &common))
	case "json":
		b = printer.JSON(result, &printer.JSONOption{
			Indent:     Options.Indent,
			HideDetail: Options.Compact,
		})
	case "html":
		b = printer.HTML(result)
	case "svg":
		b = printer.Svg(result, &printer.SvgOption{
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

	if Options.Web {
		webui.HostServer(b, Options.Listen)

		url := utils.GetUrlFromListen(Options.Listen)

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

	if Options.Output != "" {
		err := os.WriteFile(Options.Output, b, 0644)
		if err != nil {
			utils.FatalError(err)
		}
	} else {
		fmt.Println(string(b))
	}
}
