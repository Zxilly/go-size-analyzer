package web

import (
	"net/http"
)

func getHTMLHandler(content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(content))
	}
}

func HostServer(content []byte, listen string) *http.Server {
	server := &http.Server{
		Addr:    listen,
		Handler: getHTMLHandler(content),
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	return server
}
