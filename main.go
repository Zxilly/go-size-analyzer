package main

import (
	analyzer "github.com/Zxilly/go-size-analyzer/pkg"
	"github.com/spf13/cobra"
	"log"
)

var cmd = &cobra.Command{
	Use:   "go-size-view [file]",
	Short: "A tool for analysing the size of dependencies in compiled Golang binaries.",
	Long:  "A tool for analysing the size of dependencies in compiled Golang binaries, providing insight into their impact on the final build.",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

	RunE: execute,
}

func execute(_ *cobra.Command, args []string) error {
	path := args[0]

	return analyzer.Analyze(path)
}

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
