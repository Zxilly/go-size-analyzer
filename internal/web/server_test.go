package web_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/web"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestHostServer(t *testing.T) {
	content := []byte("test content")
	listen := "localhost:8080"

	server := web.HostServer(content, listen)

	assert.NotNil(t, server)
	assert.Equal(t, listen, server.Addr)

	// Send a test request to the server
	req, err := http.NewRequest("GET", utils.GetUrlFromListen(listen), nil)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	server.Handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test content", resp.Body.String())

	// Stop the server
	server.Close()
}
