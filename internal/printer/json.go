//go:build !js && !wasm

package printer

import (
	"io"
	"log/slog"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

type JSONOption struct {
	Writer     io.Writer
	Indent     *int
	HideDetail bool
}

func JSON(r *result.Result, options *JSONOption) error {
	slog.Info("JSON encoding...")

	jsonOptions := []json.Options{
		json.DefaultOptionsV2(),
		json.Deterministic(true),
	}
	if options.Indent != nil {
		jsonOptions = append(jsonOptions, jsontext.WithIndent(strings.Repeat(" ", *options.Indent)))
	}
	if options.HideDetail {
		jsonOptions = append(jsonOptions, json.WithMarshalers(marshaler.GetFileCompactMarshaler()))
	}

	err := json.MarshalWrite(options.Writer, r, jsonOptions...)

	slog.Info("JSON encoded")

	return err
}
