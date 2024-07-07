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
	s.Run("cache exist", func() {
		cacheFile, err := getCacheFilePath()
		s.Require().NoError(err)

		_, err = updateCache(cacheFile)
		s.Require().NoError(err)

		got := GetTemplate()
		s.Contains(got, constant.ReplacedStr)
	})

	s.Run("cache not exist", func() {
		cacheFile, err := getCacheFilePath()
		s.Require().NoError(err)

		err = os.Remove(cacheFile)
		s.Require().NoError(err)

		got := GetTemplate()
		s.Contains(got, constant.ReplacedStr)
	})
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}
