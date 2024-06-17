//go:build !js && !wasm

package webui_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

func TestHostServer(t *testing.T) {
	content := []byte("test content")
	listen := "127.0.0.1:8080"

	l := webui.HostServer(content, listen)
	defer func(l io.Closer) {
		_ = l.Close()
	}(l)
	assert.NotNil(t, l)

	// wait for the server to start
	time.Sleep(1 * time.Second)
	// Send a test request to the server
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
