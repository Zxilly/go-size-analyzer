package main

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

var cmd = &cobra.Command{
	Use:     "go-size-view [file]",
	Short:   "A tool for analysing the size of dependencies in compiled Golang binaries.",
	Long:    "A tool for analysing the size of dependencies in compiled Golang binaries, providing insight into their impact on the final build.",
	Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run:     execute,
	Version: Version,
}

var verbose *bool
var format *string

var hideSections *bool
var hideMain *bool
var hideStd *bool

var jsonIndent *int

var output *string

func init() {
	verbose = cmd.Flags().Bool("verbose", false, "verbose output")
	format = cmd.Flags().StringP("format", "f", "text", "output format (text, json, html)")

	hideSections = cmd.Flags().Bool("hide-sections", false, "hide sections")
	hideMain = cmd.Flags().Bool("hide-main", false, "hide main package")
	hideStd = cmd.Flags().Bool("hide-std", false, "hide standard library")

	jsonIndent = cmd.Flags().Int("indent", 0, "indentation for json output")

	output = cmd.Flags().StringP("output", "o", "", "write to file")
}

func execute(_ *cobra.Command, args []string) {
	if *verbose {
		utils.InitLogger(slog.LevelDebug)
	} else {
		utils.InitLogger(slog.LevelWarn)
	}

	path := args[0]

	result, err := internal.Analyze(path)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	option := &printer.Option{
		HideSections: *hideSections,
		HideMain:     *hideMain,
		HideStd:      *hideStd,
		Output:       *output,
		JsonIndent:   *jsonIndent,
	}

	switch *format {
	case "text":
		printer.PrintResult(result, option)
	case "json":
		printer.JsonResult(result, option)
	case "html":
	default:
		slog.Error(fmt.Sprintf("Invalid format: %s", *format))
		os.Exit(1)
	}

}

func main() {
	err := cmd.Execute()
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
}
