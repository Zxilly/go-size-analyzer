package ui_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/ui"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	got := ui.GetTemplate()

	// Should contain printer.ReplacedStr
	assert.Contains(t, got, printer.ReplacedStr)

	// Should html
	_, err := html.Parse(strings.NewReader(got))
	assert.NoError(t, err)
}
