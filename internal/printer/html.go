package printer

import (
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/ui"
	"strings"
)

const ReplacedStr = `"GSA_PACKAGE_DATA"`

func Html(r *internal.Result) []byte {
	json := Json(r, &JsonOption{HideDetail: true})
	return []byte(strings.Replace(ui.GetTemplate(), ReplacedStr, string(json), 1))
}
