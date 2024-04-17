package printer

import (
	"encoding/json"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/global"
	"log/slog"
	"strings"
)

func Json(r *internal.Result, options *JsonOption) []byte {
	if options.minify {
		global.ShowFileSizes = true
	}

	var b []byte
	var err error
	if options.Indent == nil {
		b, err = json.Marshal(r)
	} else {
		b, err = json.MarshalIndent(r, "", strings.Repeat(" ", *options.Indent))
	}
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
	}

	return b
}
