package utils

import (
	"debug/pe"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

func GetFileSize(file *os.File) uint64 {
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	return uint64(fileInfo.Size())
}

func GetImageBase(file *pe.File) uint64 {
	switch hdr := file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return uint64(hdr.ImageBase)
	case *pe.OptionalHeader64:
		return hdr.ImageBase
	default:
		panic("unknown optional header type")
	}
}

// PrefixToPath is the inverse of PathToPrefix, replacing escape sequences with
// the original character.
// from src/cmd/internal/objabi/path.go
func PrefixToPath(s string) (string, error) {
	percent := strings.IndexByte(s, '%')
	if percent == -1 {
		return s, nil
	}

	p := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] != '%' {
			p = append(p, s[i])
			i++
			continue
		}
		if i+2 >= len(s) {
			// Not enough characters remaining to be a valid escape
			// sequence.
			return "", fmt.Errorf("malformed prefix %q: escape sequence must contain two hex digits", s)
		}

		b, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
		if err != nil {
			// Not a valid escape sequence.
			return "", fmt.Errorf("malformed prefix %q: escape sequence %q must contain two hex digits", s, s[i:i+3])
		}

		p = append(p, byte(b))
		i += 3
	}
	return string(p), nil
}

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// remove time
			if a.Key == "time" {
				return slog.Attr{}
			}
			return a
		},
	})))
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

var Stdout = &SyncOutput{
	Mutex:  sync.Mutex{},
	output: os.Stdout,
}

var _ io.Writer = Stdout
