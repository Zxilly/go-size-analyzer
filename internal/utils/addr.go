package utils

import (
	"log/slog"
	"os"
	"strings"
)

func GetUrlFromListen(listen string) string {
	// get port from listen
	parts := strings.Split(listen, ":")
	if parts == nil || len(parts) < 2 {
		slog.Error("invalid listen address", "listen", listen)
		os.Exit(1)
	}
	return "http://localhost:" + parts[1]
}
