package internal

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/goretk/gore"
	"log/slog"
)

type Options struct {
	HideDisasmProgress bool

	SkipSymbol bool
	SkipDisasm bool
}

func Analyze(path string, options Options) (*Result, error) {
	file, err := gore.Open(path)
	if err != nil {
		return nil, err
	}

	k := &KnownInfo{
		Size:      utils.GetFileSize(file.GetFile()),
		BuildInfo: file.BuildInfo,

		gore:    file,
		wrapper: wrapper.NewWrapper(file.GetParsedFile()),
	}
	k.KnownAddr = entity.NewKnownAddr()
	k.UpdateVersionFlag()

	k.LoadSectionMap()

	err = k.LoadPackages()
	if err != nil {
		return nil, err
	}

	if !options.SkipSymbol {
		err = k.AnalyzeSymbol()
		if err != nil {
			if errors.Is(err, wrapper.ErrNoSymbolTable) {
				slog.Warn("Warning: no symbol table found, this can lead to inaccurate results")
			} else {
				return nil, err
			}
		}
	}

	if !options.SkipDisasm {
		err = k.Disasm(options.HideDisasmProgress)
		if err != nil {
			return nil, err
		}
	}

	// we have collected everything, now we can calculate the size

	// first, merge all results to coverage
	k.CollectCoverage()

	// for sections
	k.CalculateSectionSize()
	// for packages
	k.CalculatePackageSize()

	return BuildResult(path, k), nil
}
