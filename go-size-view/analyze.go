package go_size_view

import (
	"github.com/goretk/gore"
	"log"
)

func Analyze(path string) error {
	file, err := gore.Open(path)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	target := &KnownInfo{}
	err = target.Collect(file)
	return err
}
