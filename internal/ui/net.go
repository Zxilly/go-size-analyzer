//go:build !embed

package ui

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

const BaseUrl = "https://github.com/Zxilly/go-size-analyzer/releases/download/ui/index.html"

func GetTemplate() string {
	slog.Info("Downloading template")
	resp, err := http.Get(BaseUrl)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	return string(body)
}
