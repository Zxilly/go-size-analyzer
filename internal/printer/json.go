package printer

import (
	"encoding/json"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"log/slog"
	"strings"
)

func removeFunctions(pkg internal.PackageMap) {
	for _, p := range pkg {
		p.Functions = nil
		removeFunctions(p.SubPackages)
	}
}

func Json(r *internal.Result, options *JsonOption) string {
	if options.HideFunctions {
		removeFunctions(r.Packages)
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
