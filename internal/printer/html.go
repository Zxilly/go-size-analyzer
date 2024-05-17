package printer

import (
	"errors"
	"io"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

const ReplacedStr = `"GSA_PACKAGE_DATA"`

var ErrTemplateInvalid = errors.New("template invalid")

func HTML(r *result.Result, writer io.Writer) error {
	parts := strings.Split(webui.GetTemplate(), ReplacedStr)
	if len(parts) != 2 {
		return ErrTemplateInvalid
	}

	_, err := writer.Write([]byte(parts[0]))
	if err != nil {
		return err
	}

	err = JSON(r, &JSONOption{HideDetail: true, Writer: writer})
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(parts[1]))
	return err
}
