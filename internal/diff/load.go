package diff

import (
	"github.com/go-json-experiment/json"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func autoLoadFile(name string, options internal.Options) (*commonResult, error) {
	reader, err := mmap.Open(name)
	if err != nil {
		return nil, err
	}

	r := new(commonResult)
	if utils.DetectJSON(reader) {
		err = json.UnmarshalRead(utils.NewReaderAtAdapter(reader), r)
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	fullResult, err := internal.Analyze(name,
		reader,
		uint64(reader.Len()),
		options)
	if err != nil {
		return nil, err
	}

	return fromResult(fullResult), nil
}
