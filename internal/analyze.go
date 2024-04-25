package internal

import (
	"github.com/goretk/gore"
)

type Options struct {
	HideDisasmProgress bool
}

func Analyze(path string, options Options) (*Result, error) {
	file, err := gore.Open(path)
	if err != nil {
		return nil, err
	}

	k, err := CollectKnownInfo(file, options)
	if err != nil {
		return nil, err
	}

	return BuildResult(path, k), nil
}
