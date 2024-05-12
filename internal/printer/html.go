package printer

import (
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

const ReplacedStr = `"GSA_PACKAGE_DATA"`

func HTML(r *result.Result) []byte {
	json := JSON(r, &JSONOption{HideDetail: true})
	return []byte(strings.Replace(webui.GetTemplate(), ReplacedStr, string(json), 1))
}
