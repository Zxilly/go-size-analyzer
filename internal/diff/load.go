package diff

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/go-json-experiment/json"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type DOptions struct {
	internal.Options

	OldTarget string
	NewTarget string

	Format string

	Indent *int
}

func formatAnalyzer(analyzers []string) string {
	if len(analyzers) == 0 {
		return "none"
	}

	return strings.Join(analyzers, ", ")
}

func Diff(writer io.Writer, options DOptions) error {
	oldResult, err := autoLoadFile(options.OldTarget, options.Options)
	if err != nil {
		return err
	}

	newResult, err := autoLoadFile(options.NewTarget, options.Options)
	if err != nil {
		return err
	}

	if !requireAnalyzeModeSame(oldResult, newResult) {
		slog.Warn("The analyze mode of the two files is different")
		slog.Warn(fmt.Sprintf("%s: %s", options.NewTarget, formatAnalyzer(newResult.Analyzers)))
		slog.Warn(fmt.Sprintf("%s: %s", options.OldTarget, formatAnalyzer(oldResult.Analyzers)))
		return errors.New("analyze mode is different")
	}

	diff := newDiffResult(newResult, oldResult)

	switch options.Format {
	case "json":
		return printer.JSON(&diff, writer, &printer.JSONOption{
			Indent: nil,
		})
	case "text":
		return text(&diff, writer)
	default:
		return fmt.Errorf("format %s is not supported in diff mode", options.Format)
	}
}

func autoLoadFile(name string, options internal.Options) (*commonResult, error) {
	reader, err := mmap.Open(name)
	if err != nil {
		return nil, err
	}
	defer func(reader *mmap.ReaderAt) {
		err = reader.Close()
		if err != nil {
			slog.Warn("failed to close file", "error", err)
		}
	}(reader)

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
