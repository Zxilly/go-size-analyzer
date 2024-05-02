package server

import (
	"io"
	"net/http"
)

func HostServer(content []byte, listen string) io.Closer {
	server := &http.Server{
		Addr: listen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Server", "go-size-analyzer")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			_, _ = w.Write(content)
		}),
	}
	server.SetKeepAlivesEnabled(false)
	go func() {
		_ = server.ListenAndServe()
	}()
	return server
}
