package web

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
)

func getHTMLHandler(content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(content)
	}
}

func HostServer(content []byte, listen string) *http.Server {
	server := &http.Server{
		Addr:    listen,
		Handler: getHTMLHandler(content),
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("Failed to start server", "error", err)
				os.Exit(1)
			}
		}
	}()

	return server
}
