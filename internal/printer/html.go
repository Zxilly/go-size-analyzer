package printer

import (
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
	"strings"
)

const ReplacedStr = `"GSA_PACKAGE_DATA"`

func Html(r *result.Result) []byte {
	json := Json(r, &JsonOption{HideDetail: true})
	return []byte(strings.Replace(webui.GetTemplate(), ReplacedStr, string(json), 1))
}
