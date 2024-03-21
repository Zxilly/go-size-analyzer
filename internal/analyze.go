package internal

import (
	"github.com/goretk/gore"
)

func Analyze(path string) (*Result, error) {
	file, err := gore.Open(path)
	if err != nil {
		return nil, err
	}

	k, err := analyze(file)
	if err != nil {
		return nil, err
	}

	return BuildResult(path, k), nil
}
