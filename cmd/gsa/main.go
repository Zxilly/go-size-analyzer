package main

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/web"
	"github.com/pkg/browser"
	"log/slog"
	"os"
)

func main() {
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	if options.Verbose {
		utils.InitLogger(slog.LevelDebug)
	} else {
		utils.InitLogger(slog.LevelWarn)
	}

	if options.Version {
		PrintVersionAndExit()
	}

	result, err := internal.Analyze(options.Arg.Binary, internal.Options{
		HideDisasmProgress: options.HideProgress,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	if options.HtmlOptions.Web {
		if options.Format != "" && options.Format != "html" {
			slog.Warn("set --web option will override format to html")
		}

		options.Format = "html"
	}

	var b []byte
	switch options.Format {
	case "text", "":
		b = []byte(printer.Text(result, &printer.TextOption{
			HideSections: options.TextOptions.HideSections,
			HideMain:     options.TextOptions.HideMain,
			HideStd:      options.TextOptions.HideStd,
		}))
	case "json":
		b = printer.Json(result, &printer.JsonOption{
			Indent: options.JsonOptions.Indent,
		})
	case "html":
		b = printer.Html(result)
	default:
		slog.Error(fmt.Sprintf("Invalid format: %s", options.Format))
		os.Exit(1)
	}

	if options.Format == "html" {
		if options.HtmlOptions.Web {
			go web.HostServer(string(b), options.HtmlOptions.Listen)

			url := utils.GetUrlFromListen(options.HtmlOptions.Listen)

			fmt.Println("Server started at", url)

			if options.HtmlOptions.Open {
				err = browser.OpenURL(url)
				if err != nil {
					slog.Warn(fmt.Sprintf("Failed to open: %v", err))
				}
			}

			utils.WaitSignal()
			return
		}
	}

	if options.Output != "" {
		err := os.WriteFile(options.Output, b, 0644)
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	} else {
		fmt.Println(string(b))
	}
}
