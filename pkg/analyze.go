package pkg

import (
	"github.com/goretk/gore"
	"log"
)

func Analyze(path string) error {
	file, err := gore.Open(path)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	err = Collect(file)

	return err
}
