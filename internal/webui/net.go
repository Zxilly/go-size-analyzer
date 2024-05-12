//go:build !embed

package webui

import (
	"io"
	"log/slog"
	"net/http"

	gsv "github.com/Zxilly/go-size-analyzer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

const BaseURL = "https://github.com/Zxilly/go-size-analyzer/releases/download/ui-v" +
	gsv.StaticVersion +
	"/index.html"

func GetTemplate() string {
	slog.Info("Downloading template")
	resp, err := http.Get(BaseURL)
	if err != nil {
		utils.FatalError(err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.FatalError(err)
	}

	return string(body)
}
