//go:build !js && !wasm

package printer

import (
	"errors"
	"io"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/constant"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

var ErrTemplateInvalid = errors.New("template invalid")

func HTML(r *result.Result, writer io.Writer) error {
	parts := strings.Split(webui.GetTemplate(), constant.ReplacedStr)
	if len(parts) != 2 {
		return ErrTemplateInvalid
	}

	_, err := writer.Write([]byte(parts[0]))
	if err != nil {
		return err
	}

	err = JSON(r, writer, &JSONOption{HideDetail: true})
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(parts[1]))
	return err
}
