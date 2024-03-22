package internal

import (
	"errors"
	"github.com/goretk/gore"
	"log/slog"
)

func analyze(file *gore.GoFile) (*KnownInfo, error) {
	k := NewKnownInfo(file)

	k.LoadSectionMap()

	err := k.LoadPackages(file)
	if err != nil {
		return nil, err
	}

	err = k.AnalyzeSymbol(file)
	if err != nil {
		if errors.Is(err, ErrNoSymbolTable) {
			slog.Warn("Warning: no symbol table found, this can lead to inaccurate results")
		} else {
			return nil, err
		}
	}

	err = k.Disasm()
	if err != nil {
		return nil, err
	}

	// we have collected everything, now we can calculate the size

	// first, merge all results to coverage
	k.CollectCoverage()

	// for sections
	k.CalculateSectionSize()
	// for packages
	k.CalculatePackageSize()

	return k, nil
}
