package utils

import (
	"debug/pe"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"go4.org/intern"
	"golang.org/x/net/publicsuffix"
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

func Deduplicate(s string) string {
	return intern.GetByString(s).Get().(string) //nolint: revive // should be replaced with unique after 1.23
}

// UglyGuess an ugly hack for a known issue about golang compiler
// sees https://github.com/golang/go/issues/66313
func UglyGuess(s string) string {
	if s == "" {
		return ""
	}

	// find all parts
	parts := strings.Split(s, "/")

	result := make([]string, 0, len(parts))

	ignorePart := 1
	if strings.HasPrefix(s, "vendor/") {
		ignorePart = 2
	}

	addPart := func(p string) {
		if p != "" {
			result = append(result, p)
		}
	}

	// if any part contains more than 2 dots, we assume it's a receiver
	for i, part := range parts {
		if i < ignorePart {
			addPart(part)
			continue
		}

		pointCnt := strings.Count(part, ".")

		if pointCnt >= 2 {
			if pointCnt == 2 {
				_, icann := publicsuffix.PublicSuffix(part)
				if icann {
					addPart(part)
					continue
				}
			}

			addPart(strings.Split(part, ".")[0])
			break
		}
		addPart(part)
	}

	s = strings.Join(result, "/")
	ns, err := PrefixToPath(s)
	if err != nil {
		slog.Warn("failed to convert prefix to path", "error", err, "prefix", ns)
		return ""
	}

	return ns
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
