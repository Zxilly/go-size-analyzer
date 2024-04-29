package utils

import (
	"fmt"
	"strings"
)

func GetUrlFromListen(listen string) string {
	// get port from listen
	parts := strings.Split(listen, ":")
	if parts == nil || len(parts) < 2 {
		FatalError(fmt.Errorf("invalid listen address: %s", listen))
	}
	return "http://localhost:" + parts[1]
}
