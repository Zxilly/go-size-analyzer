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

	Imports bool
}

func Analyze(name string, reader io.ReaderAt, size uint64, options Options) (*result.Result, error) {
	slog.Info("Parsing binary...")

	file, err := gore.OpenReader(reader)
	if err != nil {
		return nil, err
	}

	slog.Info("Parsed binary")
	utils.WaitDebugger("Parsed binary")

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

	isWasm := file.FileInfo.Arch == "wasm"

	if isWasm {
		// buildinfo didn't support wasm yet, we do parse ourselves
		k.BuildInfo = &gore.BuildInfo{
			ModInfo: k.Wrapper.(*wrapper.WasmWrapper).GetModInfo(),
		}
	}

	slog.Info("Found build info")
	utils.WaitDebugger("Found build info")

	// wasm section is different and not use addr space
	// we handle it later
	if !isWasm {
		err = k.LoadSectionMap()
		if err != nil {
			return nil, err
		}

		k.KnownAddr = entity.NewKnownAddr(k.Sects)
	}

	err = k.LoadGoreInfo(file, isWasm)
	if err != nil {
		return nil, err
	}

	dwarfOk := false
	// fixme: add wasm dwarf support
	if !options.SkipDwarf && !isWasm {
		slog.Info("Parsing DWARF...")
		dwarfOk = k.TryLoadDwarf()

		if !dwarfOk {
			slog.Warn("DWARF parsing failed, fallback to symbol and disasm")
		} else {
			analyzers = append(analyzers, entity.AnalyzerDwarf)
			slog.Info("Parsed DWARF")
		}
	}

	// DWARF can still add new package, so we defer this
	k.Deps.FinishLoad(options.Imports)
	utils.WaitDebugger("DWARF and deps done")

	// we force a gc here, since the gore file is no longer used
	debug.FreeOSMemory()
	utils.WaitDebugger("After force gc")

	// fixme: add data symbol support to go gc
	if !isWasm {
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
	}

	if !options.SkipDisasm && !isWasm {
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

	if !isWasm {
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
	}

	// for packages
	k.CalculatePackageSize()

	var sections []*entity.Section
	if !isWasm {
		sections = utils.Collect(maps.Values(k.Sects.Sections))
	} else {
		codeSectUsed := uint64(0)
		_ = k.Deps.Trie.Walk(func(_ string, pkg *entity.Package) error {
			for f := range pkg.Functions {
				codeSectUsed += f.CodeSize
			}
			return nil
		})

		sections = k.Wrapper.(*wrapper.WasmWrapper).GetSections(codeSectUsed)
	}

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
