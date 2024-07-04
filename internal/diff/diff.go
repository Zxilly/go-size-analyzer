package diff

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
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

func requireAnalyzeModeSame(oldResult, newResult *commonResult) bool {
	oldModes := utils.NewSet[entity.Analyzer]()
	for _, v := range oldResult.Analyzers {
		oldModes.Add(v)
	}

	newModes := utils.NewSet[entity.Analyzer]()
	for _, v := range newResult.Analyzers {
		newModes.Add(v)
	}

	return oldModes.Equals(newModes)
}
