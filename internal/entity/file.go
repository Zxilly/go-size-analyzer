package entity

import (
	"encoding/json"
	"github.com/Zxilly/go-size-analyzer/internal/global"
)

type File struct {
	FilePath  string
	Functions []*Function

	pkg *Package
}

func (f *File) MarshalJSON() ([]byte, error) {
	if global.HideDetail {
		size := uint64(0)
		for _, fn := range f.Functions {
			size += fn.Size()
		}
		return json.Marshal(struct {
			FilePath string `json:"file_path"`
			Size     uint64 `json:"size"`
		}{
			FilePath: f.FilePath,
			Size:     size,
		})
	} else {
		return json.Marshal(struct {
			FilePath  string      `json:"file_path"`
			Functions []*Function `json:"functions"`
		}{
			FilePath:  f.FilePath,
			Functions: f.Functions,
		})
	}
}
