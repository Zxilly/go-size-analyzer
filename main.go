package main

import (
	"fmt"
	"log/slog"
	"os"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
}
