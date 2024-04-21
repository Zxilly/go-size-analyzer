package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func HostServer(content string, listen string) {
	// return string as html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(content))
	})
	err := http.ListenAndServe(listen, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
}
