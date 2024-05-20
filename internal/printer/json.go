package printer

import (
	"io"
	"log/slog"
	"strings"

	"github.com/goccy/go-json"

	"github.com/Zxilly/go-size-analyzer/internal/global"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

type JSONOption struct {
	Writer     io.Writer
	Indent     *int
	HideDetail bool
}

func JSON(r *result.Result, options *JSONOption) error {
	if options.HideDetail {
		global.HideDetail = true
	}

	slog.Info("JSON encoding...")

	encoder := json.NewEncoder(options.Writer)
	if options.Indent != nil {
		encoder.SetIndent("", strings.Repeat(" ", *options.Indent))
	}
	err := encoder.Encode(r)

	slog.Info("JSON encode done")

	return err
}
