package printer

import (
	"encoding/json"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"log/slog"
	"os"
	"strings"
)

func JsonResult(r *internal.Result, options *Option) {
	if options.HideStd {
		slog.Warn("hide std is a no-op for json output")
	}
	if options.HideMain {
		slog.Warn("hide main is a no-op for json output")
	}
	if options.HideSections {
		slog.Warn("hide sections is a no-op for json output")
	}

	s, err := json.MarshalIndent(r, "", strings.Repeat(" ", options.JsonIndent))
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
	}

	if options.Output == "" {
		fmt.Println(string(s))
	} else {
		err = os.WriteFile(options.Output, s, 0644)
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	}
}
