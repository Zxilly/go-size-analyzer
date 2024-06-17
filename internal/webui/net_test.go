//go:build !embed && !js && !wasm

package webui

import (
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Zxilly/go-size-analyzer/internal/constant"
)

func TestUpdateCacheFlag(t *testing.T) {
	var option struct {
		Flag UpdateCacheFlag `help:"Update cache"`
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

type CacheSuite struct {
	suite.Suite
}

func (s *CacheSuite) TestGetTemplate() {
	s.T().Run("cache exist", func(t *testing.T) {
		cacheFile, err := getCacheFilePath()
		require.NoError(t, err)

		_, err = updateCache(cacheFile)
		require.NoError(t, err)

		got := GetTemplate()
		assert.Contains(t, got, constant.ReplacedStr)
	})

	s.T().Run("cache not exist", func(t *testing.T) {
		cacheFile, err := getCacheFilePath()
		require.NoError(t, err)

		err = os.Remove(cacheFile)
		require.NoError(t, err)

		got := GetTemplate()
		assert.Contains(t, got, constant.ReplacedStr)
	})
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}
