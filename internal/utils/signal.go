package utils

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// WaitSignal waits for a Ctrl+C signal to exit the program.
func WaitSignal() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Press Ctrl+C to exit")
	<-done
}
