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

	isWasm := file.FileInfo.Arch == "wasm"

	if isWasm {
		// buildinfo didn't support wasm yet, we do parse ourselves
		k.BuildInfo = &gore.BuildInfo{
			ModInfo: k.Wrapper.(*wrapper.WasmWrapper).GetModInfo(),
		}
	}

	slog.Info("Found build info")
	utils.WaitDebugger("Found build info")

	if err = k.LoadSectionMap(); err != nil {
		return nil, err
	}

	k.KnownAddr = entity.NewKnownAddr(k.Sects)

	if err = k.LoadGoreInfo(file, isWasm); err != nil {
		return nil, err
	}

	var sections []*entity.Section
	var analyzers []entity.Analyzer
	if isWasm {
		sections, analyzers, err = analyzeWasm(k, options)
	} else {
		sections, analyzers, err = analyzeNative(k, options)
	}
	if err != nil {
		return nil, err
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

// runOptionalAnalyzer runs fn and appends tag to analyzers on success.
// If fn returns ErrNoGoVersionFound the error is logged and skipped.
func runOptionalAnalyzer(
	fn func() error,
	tag entity.Analyzer,
	analyzers *[]entity.Analyzer,
	warnMsg string,
) error {
	if err := fn(); err != nil {
		if errors.Is(err, gore.ErrNoGoVersionFound) {
			slog.Warn(warnMsg, "err", err)
			return nil
		}
		return err
	}
	*analyzers = append(*analyzers, tag)
	return nil
}

func analyzeWasm(k *knowninfo.KnownInfo, options Options) ([]*entity.Section, []entity.Analyzer, error) {
	// Gore file is fully consumed after LoadGoreInfo for wasm (no DWARF step).
	debug.FreeOSMemory()
	utils.WaitDebugger("After force gc")

	analyzers := []entity.Analyzer{entity.AnalyzerPclntab}

	if err := runOptionalAnalyzer(k.AnalyzeTypes, entity.AnalyzerTyp, &analyzers,
		"Type analysis skipped: Go version not available in binary"); err != nil {
		return nil, nil, err
	}
	if err := runOptionalAnalyzer(k.AnalyzePclntabMeta, entity.AnalyzerPclntabMeta, &analyzers,
		"Pclntab meta analysis skipped: Go version not available in binary"); err != nil {
		return nil, nil, err
	}

	// All analyzers done; materialize the package tree.
	k.Deps.FinishLoad(options.Imports)
	utils.WaitDebugger("All analyzers and deps done")
	k.Deps.ClearCaches()

	k.CalculatePackageSize()

	wasmWrapper := k.Wrapper.(*wrapper.WasmWrapper)
	codeSectUsed := wasmCodeSectUsed(k)
	dataSectUsed := wasmWrapper.ComputeDataSectUsed(k.KnownAddr.SymbolAddrSpace)
	sections := wasmWrapper.GetSections(codeSectUsed, dataSectUsed)

	return sections, analyzers, nil
}

func analyzeNative(k *knowninfo.KnownInfo, options Options) ([]*entity.Section, []entity.Analyzer, error) {
	analyzers := []entity.Analyzer{entity.AnalyzerPclntab}

	// fixme: add wasm dwarf support
	if !options.SkipDwarf {
		slog.Info("Parsing DWARF...")
		if k.TryLoadDwarf() {
			analyzers = append(analyzers, entity.AnalyzerDwarf)
			slog.Info("Parsed DWARF")
		} else {
			slog.Warn("DWARF parsing failed, fallback to symbol and disasm")
		}
	}

	// Gore file is fully consumed after DWARF parsing.
	debug.FreeOSMemory()
	utils.WaitDebugger("After force gc")

	// fixme: add data symbol support to go gc
	record := !options.SkipSymbol
	if err := k.AnalyzeSymbol(record); err != nil {
		if !errors.Is(err, wrapper.ErrNoSymbolTable) {
			return nil, nil, err
		}
		slog.Warn("No symbol table found, this can lead to inaccurate results")
	}
	if record {
		analyzers = append(analyzers, entity.AnalyzerSymbol)
	}
	utils.WaitDebugger("Symbol done")

	if err := runOptionalAnalyzer(k.AnalyzeTypes, entity.AnalyzerTyp, &analyzers,
		"Type analysis skipped: Go version not available in binary"); err != nil {
		return nil, nil, err
	}
	if err := runOptionalAnalyzer(k.AnalyzePclntabMeta, entity.AnalyzerPclntabMeta, &analyzers,
		"Pclntab meta analysis skipped: Go version not available in binary"); err != nil {
		return nil, nil, err
	}

	if !options.SkipDisasm {
		if k.GoStringSymbol == nil {
			slog.Info("no go:string.* symbol found, false-positive rates may rise")
		}

		if err := k.Disasm(); err != nil {
			return nil, nil, err
		}
		analyzers = append(analyzers, entity.AnalyzerDisasm)
	}

	// DWARF, symbol, type, pclntab-meta, and disasm analyzers can all create
	// new packages, so materialize the package tree only after they have all completed.
	k.Deps.FinishLoad(options.Imports)
	utils.WaitDebugger("All analyzers and deps done")
	k.Deps.ClearCaches()

	if err := k.CollectCoverage(); err != nil {
		return nil, nil, err
	}
	if err := k.CalculateSectionSize(); err != nil {
		return nil, nil, err
	}
	k.CalculatePackageSize()

	sections := utils.Collect(maps.Values(k.Sects.Sections))
	return sections, analyzers, nil
}

// wasmCodeSectUsed sums the code size of all functions across all packages,
// used to compute KnownSize for the Wasm code section.
func wasmCodeSectUsed(k *knowninfo.KnownInfo) uint64 {
	total := uint64(0)
	_ = k.Deps.Trie.Walk(func(_ string, pkg *entity.Package) error {
		for f := range pkg.Functions {
			total += f.CodeSize
		}
		return nil
	})
	return total
}
