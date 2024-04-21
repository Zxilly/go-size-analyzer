//go:build embed

package ui

import (
	_ "embed"
)

//go:embed index.html
var tmpl string

func GetTemplate() string {
	return tmpl
}
