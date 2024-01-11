package main

import (
	go_size_view "github.com/Zxilly/go-size-view/go-size-view"
	"github.com/goretk/gore"
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

func execute(cmd *cobra.Command, args []string) {
	path := args[0]

	file, err := gore.Open(path)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	go_size_view.Analyze(file)

}

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
