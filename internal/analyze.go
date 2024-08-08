package internal

import (
	"cmp"
	"errors"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"runtime/debug"
	"slices"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/knowninfo"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
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

	slog.Info("Parsed binary done")
	utils.WaitDebugger("Parsed binary done")

	slog.Info("Finding build info...")

	k := &knowninfo.KnownInfo{
		Size:      size,
		BuildInfo: file.BuildInfo,

		Gore:    file,
		Wrapper: wrapper.NewWrapper(file.GetParsedFile()),
	}

	analyzers := []entity.Analyzer{
		entity.AnalyzerPclntab,
	}

	slog.Info("Found build info")
	utils.WaitDebugger("Found build info")

	err = k.LoadSectionMap()
	if err != nil {
		return nil, err
	}

	k.KnownAddr = entity.NewKnownAddr(k.Sects)

	err = k.LoadGoreInfo(file)
	if err != nil {
		return nil, err
	}

	dwarfOk := false
	if !options.SkipDwarf {
		slog.Info("Parsing DWARF...")
		dwarfOk = k.TryLoadDwarf()
	}

	if !dwarfOk && !options.SkipDwarf {
		slog.Warn("DWARF parsing failed, fallback to symbol and disasm")
	}

	if dwarfOk {
		analyzers = append(analyzers, entity.AnalyzerDwarf)
		slog.Info("Parsed DWARF")
	}

	// DWARF can still add new package, so we defer this
	k.Deps.FinishLoad()
	utils.WaitDebugger("DWARF and deps done")

	// we force a gc here, since the gore file is no longer used
	debug.FreeOSMemory()
	utils.WaitDebugger("After force gc")

	record := !dwarfOk && !options.SkipSymbol
	err = k.AnalyzeSymbol(record)
	if err != nil {
		if !errors.Is(err, wrapper.ErrNoSymbolTable) {
			return nil, err
		}
		slog.Warn("No symbol table found, this can lead to inaccurate results")
	}
	if record {
		analyzers = append(analyzers, entity.AnalyzerSymbol)
	}
	utils.WaitDebugger("Symbol done")

	if !options.SkipDisasm {
		if k.GoStringSymbol == nil {
			slog.Info("no go:string.* symbol found, false-positive rates may rise")
		}

		err = k.Disasm()
		if err != nil {
			return nil, err
		}
		analyzers = append(analyzers, entity.AnalyzerDisasm)
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

	sections := utils.Collect(maps.Values(k.Sects.Sections))
	slices.SortFunc(sections, func(a, b *entity.Section) int {
		return cmp.Compare(a.Name, b.Name)
	})
	slices.Sort(analyzers)

	utils.WaitDebugger("Analyze done")

	return &result.Result{
		Name:      filepath.Base(name),
		Size:      k.Size,
		Packages:  k.Deps.TopPkgs,
		Sections:  sections,
		Analyzers: analyzers,
	}, nil
}
