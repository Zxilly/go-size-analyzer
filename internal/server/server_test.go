package server_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/server"
	"net/http"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestHostServer(t *testing.T) {
	content := []byte("test content")
	listen := "localhost:8080"

	l := server.HostServer(content, listen)
	defer l.Close()
	assert.NotNil(t, l)

	// Send a test request to the server
	req, err := http.NewRequest("GET", utils.GetUrlFromListen(listen), nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
