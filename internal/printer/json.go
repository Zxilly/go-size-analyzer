//go:build !js && !wasm

package printer

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"log/slog"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
)

type JSONOption struct {
	Indent        *int
	HideDetail    bool
	EscapeForHTML bool
}

func JSON(r any, writer io.Writer, options *JSONOption) error {
	slog.Info("JSON encoding...")

	jsonOptions := []json.Options{
		json.DefaultOptionsV2(),
		json.Deterministic(true),
		jsontext.EscapeForHTML(options.EscapeForHTML),
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
