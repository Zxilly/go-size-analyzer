package main

import (
	analyzer "github.com/Zxilly/go-size-analyzer/internal"
	"github.com/spf13/cobra"
	"log"
)

var cmd = &cobra.Command{
	Use:   "go-size-view [file]",
	Short: "A tool for analysing the size of dependencies in compiled Golang binaries.",
	Long:  "A tool for analysing the size of dependencies in compiled Golang binaries, providing insight into their impact on the final build.",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run:   execute,
}

func execute(_ *cobra.Command, args []string) {
	path := args[0]

	err := analyzer.Analyze(path)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}