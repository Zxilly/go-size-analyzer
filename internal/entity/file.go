package entity

import (
	"encoding/json"
	"github.com/Zxilly/go-size-analyzer/internal/global"
	"golang.org/x/exp/maps"
)

type File struct {
	FilePath  string
	Functions []*Function

	Pkg *Package
}

func (f *File) MarshalJSON() ([]byte, error) {
	if !global.UseMinifyFormatForFunc {
		return json.Marshal(f)
	}

	size := uint64(0)
	names := make(map[string]struct{})
	for _, fn := range f.Functions {
		size += fn.Size

		name := fn.Name
		if fn.Type == FuncTypeMethod {
			name = fn.Receiver + "." + name
		}
		names[name] = struct{}{}
	}

	return json.Marshal(struct {
		FilePath  string   `json:"file_path"`
		Size      uint64   `json:"size"`
		Functions []string `json:"functions"`
	}{
		FilePath:  f.FilePath,
		Size:      size,
		Functions: maps.Keys(names),
	})
}
