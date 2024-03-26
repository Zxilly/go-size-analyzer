package printer

import (
	"encoding/json"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/global"
	"log/slog"
	"strings"
)

func Json(r *internal.Result, options *JsonOption) string {
	if options.minify {
		global.UseMinifyFormatForFunc = true
	}

	var s []byte
	var err error
	if options.Indent == nil {
		s, err = json.Marshal(r)
	} else {
		s, err = json.MarshalIndent(r, "", strings.Repeat(" ", *options.Indent))
	}
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
	}

	return string(s)
}
