package internal

import (
	"errors"
	"log/slog"
	"path"
	"runtime"

	"github.com/ZxillyFork/gore"
	"golang.org/x/exp/maps"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type Options struct {
	SkipSymbol bool
	SkipDisasm bool
}

func Analyze(bin string, options Options) (*result.Result, error) {
	file, err := gore.Open(bin)
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

	// collect for load and possible disassemble in go version read
	runtime.GC()

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

	// collect for package and symbol
	runtime.GC()

	if !options.SkipDisasm {
		err = k.Disasm()
		if err != nil {
			return nil, err
		}
	}

	// collect for disassemble
	// which can be very memory consuming
	runtime.GC()

	// we have collected everything, now we can calculate the size

	// first, merge all results to coverage
	k.CollectCoverage()

	// for sections
	k.CalculateSectionSize()
	// for packages
	k.CalculatePackageSize()

	return &result.Result{
		Name:     path.Base(bin),
		Size:     k.Size,
		Packages: k.Deps.TopPkgs,
		Sections: maps.Values(k.Sects.Sections),
	}, nil
}
