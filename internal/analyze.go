package internal

import (
	"encoding/json"
	"github.com/goretk/gore"
	"os"
)

func Analyze(path string) error {
	file, err := gore.Open(path)
	if err != nil {
		return err
	}

	k, err := analyze(file)
	if err != nil {
		return err
	}

	r := BuildResult(path, k)

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	os.WriteFile("result.json", b, 0644)

	return nil
}
