package printer

import (
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

const ReplacedStr = `"GSA_PACKAGE_DATA"`

func Html(r *result.Result) []byte {
	json := Json(r, &JsonOption{HideDetail: true})
	return []byte(strings.Replace(webui.GetTemplate(), ReplacedStr, string(json), 1))
}
