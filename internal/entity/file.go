package entity

import (
	"encoding/json"

	"github.com/Zxilly/go-size-analyzer/internal/global"
)

type File struct {
	FilePath  string      `json:"file_path"`
	Functions []*Function `json:"functions"`

	Pkg *Package `json:"-"`
}

func (f *File) FullSize() uint64 {
	size := uint64(0)
	for _, fn := range f.Functions {
		size += fn.Size()
	}
	return size
}

func (f *File) PclnSize() uint64 {
	size := uint64(0)
	for _, fn := range f.Functions {
		size += fn.PclnSize.Size()
	}
	return size
}

func (f *File) MarshalJSON() ([]byte, error) {
	if global.HideDetail {
		return json.Marshal(struct {
			FilePath string `json:"file_path"`
			Size     uint64 `json:"size"`
			PclnSize uint64 `json:"pcln_size"`
		}{
			FilePath: f.FilePath,
			Size:     f.FullSize(),
			PclnSize: f.PclnSize(),
		})
	}

	return json.Marshal(struct {
		FilePath  string      `json:"file_path"`
		Functions []*Function `json:"functions"`
	}{
		FilePath:  f.FilePath,
		Functions: f.Functions,
	})
}
