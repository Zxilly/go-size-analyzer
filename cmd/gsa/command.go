package main

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/ZxillyFork/go-flags"
	"log/slog"
	"os"
	"runtime/debug"
)

type Options struct {
	Verbose bool   `long:"verbose" description:"Verbose output"`
	Format  string `short:"f" long:"format" description:"Output format" default:"text" choice:"text" choice:"json" choice:"html"`

	TextOptions struct {
		HideSections bool `long:"hide-sections" description:"Hide sections"`
		HideMain     bool `long:"hide-main" description:"Hide main package"`
		HideStd      bool `long:"hide-std" description:"Hide standard library"`
	} `group:"Text Options"`

	JsonOptions struct {
		HideFunctions bool `long:"hide-functions" description:"Hide functions field in package"`
		Indent        *int `long:"indent" description:"Indentation for json output"`
	} `group:"Json Options"`

	Output  string `short:"o" long:"output" description:"Write to file"`
	Version bool   `long:"version" description:"Show version"`

	Arg struct {
		Binary string `positional-arg-name:"file" description:"Binary file to analyze"`
	} `positional-args:"yes"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)

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

	var s string
	switch options.Format {
	case "text":
		s = printer.Text(result, &printer.TextOption{
			HideSections: options.TextOptions.HideSections,
			HideMain:     options.TextOptions.HideMain,
			HideStd:      options.TextOptions.HideStd,
		})
	case "json":
		s = printer.Json(result, &printer.JsonOption{
			Indent:        options.JsonOptions.Indent,
			HideFunctions: options.JsonOptions.HideFunctions,
		})
	case "html":
	default:
		slog.Error(fmt.Sprintf("Invalid format: %s", options.Format))
		os.Exit(1)
	}

	if options.Output != "" {
		err := os.WriteFile(options.Output, []byte(s), 0644)
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	} else {
		fmt.Println(s)
	}
}
