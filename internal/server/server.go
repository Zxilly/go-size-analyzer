package server

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"net/http"
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
				utils.FatalError(err)
			}
		}
	}()

	return server
}
