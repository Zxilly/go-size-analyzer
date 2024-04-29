//go:build !embed

package webui

import (
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"io"
	"log/slog"
	"net/http"
)

const BaseUrl = "https://github.com/Zxilly/go-size-analyzer/releases/download/ui/index.html"

func GetTemplate() string {
	slog.Info("Downloading template")
	resp, err := http.Get(BaseUrl)
	if err != nil {
		utils.FatalError(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.FatalError(err)
	}

	return string(body)
}
