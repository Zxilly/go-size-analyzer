package internal

import (
	"cmp"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/ZxillyFork/gore"
	"golang.org/x/exp/maps"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/knowninfo"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type Options struct {
	SkipSymbol bool
	SkipDisasm bool
	SkipDwarf  bool
}

func Analyze(name string, reader io.ReaderAt, size uint64, options Options) (*result.Result, error) {
	slog.Info("Parsing binary...")

	file, err := gore.Open(reader)
	if err != nil {
		return nil, err
	}

	slog.Info("Parsing binary done")
	slog.Info("Finding build info...")

	k := &knowninfo.KnownInfo{
		Size:      size,
		BuildInfo: file.BuildInfo,

		Gore:    file,
		Wrapper: wrapper.NewWrapper(file.GetParsedFile()),
	}
	k.KnownAddr = entity.NewKnownAddr()
	k.VersionFlag = k.UpdateVersionFlag()

	slog.Info("Build info found")

	err = k.LoadSectionMap()
	if err != nil {
		return nil, err
	}

	err = k.LoadPackages()
	if err != nil {
		return nil, err
	}

	dwarfOk := false
	if !options.SkipDwarf {
		dwarfOk = k.TryLoadDwarf()
	}

	if !dwarfOk && !options.SkipDwarf {
		slog.Warn("DWARF parsing failed, fallback to symbol and disasm")
	}

	// DWARF can still add new package
	k.Deps.FinishLoad()

	if !dwarfOk {
		// fallback to symbol and disasm
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
	}

	// we have collected everything, now we can calculate the size

	// first, merge all results to coverage
	err = k.CollectCoverage()
	if err != nil {
		return nil, err
	}

	// for sections
	err = k.CalculateSectionSize()
	if err != nil {
		return nil, err
	}

	// for packages
	k.CalculatePackageSize()

	sections := maps.Values(k.Sects.Sections)
	slices.SortFunc(sections, func(a, b *entity.Section) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &result.Result{
		Name:     filepath.Base(name),
		Size:     k.Size,
		Packages: k.Deps.TopPkgs,
		Sections: sections,
	}, nil
}
