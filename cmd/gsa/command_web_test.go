//go:build !wasm

package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestWebListenerDefaultsToLoopback(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.bin")
	require.NoError(t, os.WriteFile(input, nil, 0o600))
	for _, listen := range []string{"", "0.0.0.0:9090", "[::1]:9090"} {
		t.Run("listen="+listen, func(t *testing.T) {
			options := Options
			parser, err := kong.New(&options, kong.Vars{"version": "test"})
			require.NoError(t, err)
			args := []string{"--web", input}
			if listen != "" {
				args = append(args, "--listen", listen)
			}
			_, err = parser.Parse(args)
			require.NoError(t, err)
			require.True(t, options.Web)
			if listen != "" {
				require.Equal(t, listen, options.Listen)
				return
			}
			host, port, err := net.SplitHostPort(options.Listen)
			require.NoError(t, err)
			require.True(t, net.ParseIP(host).IsLoopback(), "default listener exposes the report outside loopback: %s", options.Listen)
			require.Equal(t, "8080", port)
		})
	}
}
