package webui_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/webui"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHostServer(t *testing.T) {
	content := []byte("test content")
	listen := "127.0.0.1:8080"

	l := webui.HostServer(content, listen)
	defer l.Close()
	assert.NotNil(t, l)

	// wait for the server to start
	time.Sleep(1 * time.Second)
	// Send a test request to the server
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
