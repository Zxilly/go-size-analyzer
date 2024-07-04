package diff

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-json-experiment/json"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func Diff(oldTarget, newTarget string, options internal.Options) error {
	oldResult, err := autoLoadFile(oldTarget, options)
	if err != nil {
		return err
	}

	newResult, err := autoLoadFile(newTarget, options)
	if err != nil {
		return err
	}

	if !requireAnalyzeModeSame(oldResult, newResult) {
		formatAnalyzer := func(analyzers []string) string {
			if len(analyzers) == 0 {
				return "none"
			}

			return strings.Join(analyzers, ", ")
		}

		slog.Warn("The analyze mode of the two files is different")
		slog.Warn(fmt.Sprintf("%s: %s", newTarget, formatAnalyzer(newResult.Analyzers)))
		slog.Warn(fmt.Sprintf("%s: %s", oldTarget, formatAnalyzer(oldResult.Analyzers)))
		return errors.New("analyze mode is different")
	}

	return nil
}

func autoLoadFile(name string, options internal.Options) (*commonResult, error) {
	reader, err := mmap.Open(name)
	if err != nil {
		return nil, err
	}

	r := new(commonResult)
	if utils.DetectJSON(reader) {
		err = json.UnmarshalRead(utils.NewReaderAtAdapter(reader), r)
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	fullResult, err := internal.Analyze(
		name,
		reader,
		uint64(reader.Len()),
		options)
	if err != nil {
		return nil, err
	}

	return fromResult(fullResult), nil
}
