//go:build !embed

package webui_test

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

func TestUpdateCacheFlag(t *testing.T) {
	var option struct {
		Flag webui.UpdateCacheFlag `help:"Update cache"`
	}

	exited := false

	k, err := kong.New(&option,
		kong.Name("test"),
		kong.Description("test"),
		kong.Exit(func(_ int) {
			exited = true
		}))
	require.NoError(t, err)

	_, err = k.Parse([]string{"--flag"})
	require.NoError(t, err)

	assert.True(t, exited)
}
