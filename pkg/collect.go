package pkg

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"github.com/goretk/gore"
	"log"
)

func analyze(file *gore.GoFile) (*KnownInfo, error) {
	b := &KnownInfo{}

	b.FoundAddr = NewFoundAddr()

	b.SectionMap = loadSectionMap(file)
	b.Size = tool.GetFileSize(file.GetFile())
	b.BuildInfo = file.BuildInfo

	b.updateVersionFlag()

	err := b.SectionMap.AssertSize(b.Size)
	if err != nil {
		return nil, err
	}

	pkgs, err := b.loadPackages(file)
	if err != nil {
		return nil, err
	}
	b.Packages = pkgs

	err = b.analyzeSymbol(file)
	if err != nil {
		if errors.Is(err, ErrNoSymbolTable) {
			log.Println("Warning: no symbol table found, this can lead to inaccurate results")
		} else {
			return nil, err
		}
	}

	err = b.tryDisasm(file)
	if err != nil {
		return nil, err
	}

	err = b.FoundAddr.Validate()
	if err != nil {
		return nil, err
	}

	return b, nil
}
