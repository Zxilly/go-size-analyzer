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
	"runtime/debug"
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
		fmt.Println("Version:", Version)
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("Failed to read build info")
			os.Exit(1)
		}
		for _, m := range info.Settings {
			switch m.Key {
			case "vcs.revision":
				fmt.Printf("Git revision: %s\n", m.Value)
			case "vcs.time":
				fmt.Printf("Build time: %s\n", m.Value)
			}
		}
		os.Exit(0)
	}

	result, err := internal.Analyze(options.Arg.Binary)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	var b []byte
	switch options.Format {
	case "text":
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
