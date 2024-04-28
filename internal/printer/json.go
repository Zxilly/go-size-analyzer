package printer

import (
	"encoding/json"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/global"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"log/slog"
	"strings"
)

type JsonOption struct {
	Indent     *int
	HideDetail bool
}

func Json(r *result.Result, options *JsonOption) []byte {
	if options.HideDetail {
		global.HideDetail = true
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
