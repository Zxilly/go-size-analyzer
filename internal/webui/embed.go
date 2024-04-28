//go:build embed

package webui

import (
	_ "embed"
)

//go:embed index.html
var tmpl string

func GetTemplate() string {
	return tmpl
}
