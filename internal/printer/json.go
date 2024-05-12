package printer

import (
	"encoding/json"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/global"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type JSONOption struct {
	Indent     *int
	HideDetail bool
}

func JSON(r *result.Result, options *JSONOption) []byte {
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
		utils.FatalError(err)
	}

	return b
}
