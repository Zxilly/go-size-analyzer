package webui

import (
	"io"
	"net/http"
	"time"
)

func HostServer(content []byte, listen string) io.Closer {
	server := &http.Server{
		Addr:              listen,
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
