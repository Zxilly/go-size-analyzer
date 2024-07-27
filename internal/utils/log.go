package utils

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

var startTime time.Time

func InitLogger(level slog.Level) {
	startTime = time.Now()
	slog.SetDefault(slog.New(slog.NewTextHandler(Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// remove time
			if a.Key == "time" {
				return slog.Duration(slog.TimeKey, time.Since(startTime))
			}
			return a
		},
		Level: level,
	})))
}

var exitFunc = os.Exit

func UsePanicForExit() {
	exitFunc = func(code int) {
		panic(fmt.Errorf("exit: %d", code))
	}
}

func FatalError(err error) {
	if err == nil {
		return
	}

	slog.Error(fmt.Sprintf("Fatal error: %v", err))

	exitFunc(1)
}

type SyncOutput struct {
	sync.Mutex
	output io.Writer
}

func (s *SyncOutput) Write(p []byte) (n int, err error) {
	s.Lock()
	defer s.Unlock()
	return s.output.Write(p)
}

func (s *SyncOutput) SetOutput(output io.Writer) {
	s.Lock()
	defer s.Unlock()
	s.output = output
}

var Stdout = &SyncOutput{
	Mutex:  sync.Mutex{},
	output: os.Stderr,
}

var _ io.Writer = Stdout
