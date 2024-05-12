//go:build !embed

package webui

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	gsv "github.com/Zxilly/go-size-analyzer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

var BaseURL = fmt.Sprintf("https://github.com/Zxilly/go-size-analyzer/releases/download/ui-v%d/index.html", gsv.GetStaticVersion())

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
