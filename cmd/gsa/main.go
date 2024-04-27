package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/web"
	"github.com/pkg/browser"
)

func main() {
	if Options.Verbose {
		utils.InitLogger(slog.LevelDebug)
	} else {
		utils.InitLogger(slog.LevelWarn)
	}

	result, err := internal.Analyze(Options.Binary, internal.Options{
		HideDisasmProgress: Options.HideProgress,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	if Options.Web {
		Options.Format = "html"
	}

	var b []byte
	switch Options.Format {
	case "text":
		b = []byte(printer.Text(result, &printer.TextOption{
			HideSections: Options.HideSections,
			HideMain:     Options.HideMain,
			HideStd:      Options.HideStd,
		}))
	case "json":
		b = printer.Json(result, &printer.JsonOption{
			Indent: Options.Indent,
		})
	case "html":
		b = printer.Html(result)
	default:
		slog.Error(fmt.Sprintf("Invalid format: %s", Options.Format))
		os.Exit(1)
	}

	if Options.Web {
		server := web.HostServer(b, Options.Listen)
		defer server.Close()

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
			slog.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	} else {
		fmt.Println(string(b))
	}
}
