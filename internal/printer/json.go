//go:build !js && !wasm

package printer

import (
	"io"
	"log/slog"
	"strings"

	"encoding/json/v2"
	"encoding/json/jsontext"

	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
)

type JSONOption struct {
	Indent     *int
	HideDetail bool
}

func JSON(r any, writer io.Writer, options *JSONOption) error {
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

	err := json.MarshalWrite(writer, r, jsonOptions...)

	slog.Info("JSON encoded")

	return err
}
