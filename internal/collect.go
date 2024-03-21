package internal

import (
	"errors"
	"github.com/goretk/gore"
	"log"
)

func analyze(file *gore.GoFile) (*KnownInfo, error) {
	b := NewKnownInfo(file)

	b.LoadSectionMap()

	err := b.LoadPackages(file)
	if err != nil {
		return nil, err
	}

	err = b.AnalyzeSymbol(file)
	if err != nil {
		if errors.Is(err, ErrNoSymbolTable) {
			log.Println("Warning: no symbol table found, this can lead to inaccurate results")
		} else {
			return nil, err
		}
	}

	err = b.Disasm()
	if err != nil {
		return nil, err
	}

	return b, nil
}
