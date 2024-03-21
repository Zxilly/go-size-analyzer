package main

import (
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
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

func init() {
	verbose = cmd.Flags().Bool("verbose", false, "verbose output")
	format = cmd.Flags().StringP("format", "f", "text", "output format (text, json, html)")
}

func execute(_ *cobra.Command, args []string) {
	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	} else {
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}

	path := args[0]

	result, err := internal.Analyze(path)
	if err != nil {
		slog.Error("Error: %v", err)
		os.Exit(1)
	}

	switch *format {
	case "text":
		printer.PrintResult(result)
	case "json":
	case "html":
	default:
		slog.Error("Invalid format: %s", *format)
		os.Exit(1)
	}

}

func main() {
	err := cmd.Execute()
	if err != nil {
		slog.Error("Error: %v", err)
		os.Exit(1)
	}
}
