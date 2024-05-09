//go:build !embed

package webui

import (
	"fmt"
	gsv "github.com/Zxilly/go-size-analyzer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"io"
	"log/slog"
	"net/http"
)

var BaseUrl = fmt.Sprintf("https://github.com/Zxilly/go-size-analyzer/releases/download/ui-v%d/index.html", gsv.GetStaticVersion())

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
