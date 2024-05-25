package internal

import (
	"errors"
	"io"
	"log/slog"
	"path"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/ZxillyFork/gore"
	"golang.org/x/exp/maps"
)

type Options struct {
	SkipSymbol bool
	SkipDisasm bool
}

func Analyze(name string, reader io.ReaderAt, size uint64, options Options) (*result.Result, error) {
	slog.Info("Parsing binary...")

	file, err := gore.Open(reader)
	if err != nil {
		return nil, err
	}

	slog.Info("Parsing binary done")
	slog.Info("Finding build info...")

	k := &KnownInfo{
		Size:      size,
		BuildInfo: file.BuildInfo,

		gore:    file,
		wrapper: wrapper.NewWrapper(file.GetParsedFile()),
	}
	k.KnownAddr = entity.NewKnownAddr()
	k.UpdateVersionFlag()

	slog.Info("Build info found")

	k.LoadSectionMap()

	err = k.LoadPackages()
	if err != nil {
		return nil, err
	}

	if !options.SkipSymbol {
		err = k.AnalyzeSymbol()
		if err != nil {
			if !errors.Is(err, wrapper.ErrNoSymbolTable) {
				return nil, err

			}
			slog.Warn("Warning: no symbol table found, this can lead to inaccurate results")
		}
	}

	if !options.SkipDisasm {
		err = k.Disasm()
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

	return &result.Result{
		Name:     path.Base(name),
		Size:     k.Size,
		Packages: k.Deps.TopPkgs,
		Sections: maps.Values(k.Sects.Sections),
	}, nil
}
