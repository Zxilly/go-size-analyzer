package utils

import (
	"fmt"
	"log/slog"
	"net"
)

const defaultURL = "http://localhost:8080"

func GetURLFromListen(listen string) string {
	addr, err := net.ResolveTCPAddr("tcp", listen)
	if err != nil {
		slog.Warn(fmt.Sprintf("Error resolving listen address: %v", err))
		return defaultURL
	}

	if addr.Port == 0 {
		addr.Port = 8080
	}

	return fmt.Sprintf("http://localhost:%d", addr.Port)
}
