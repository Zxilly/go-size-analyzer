package server_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/server"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestHostServer(t *testing.T) {
	content := []byte("test content")
	listen := "localhost:8080"

	s := server.HostServer(content, listen)

	assert.NotNil(t, s)
	assert.Equal(t, listen, s.Addr)

	// Send a test request to the server
	req, err := http.NewRequest("GET", utils.GetUrlFromListen(listen), nil)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	s.Handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test content", resp.Body.String())

	// Stop the server
	s.Close()
}
