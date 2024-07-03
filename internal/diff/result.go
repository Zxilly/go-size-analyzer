package diff

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

type commonResult struct {
	Size int64 `json:"size"`

	Analyzers []entity.Analyzer        `json:"analyzers"`
	Packages  map[string]commonPackage `json:"packages"`
	Sections  []commonSection          `json:"sections"`
}

type commonPackage struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type commonSection struct {
	Name string `json:"name"`

	FileSize  int64 `json:"file_size"`
	KnownSize int64 `json:"known_size"`
}

func fromResult(r *result.Result) *commonResult {
	c := commonResult{
		Size:      int64(r.Size),
		Analyzers: r.Analyzers,
		Packages:  make(map[string]commonPackage),
		Sections:  make([]commonSection, len(r.Sections)),
	}

	for k, v := range r.Packages {
		c.Packages[k] = commonPackage{
			Name: v.Name,
			Size: int64(v.Size),
		}
	}

	for i, v := range r.Sections {
		c.Sections[i] = commonSection{
			Name:      v.Name,
			FileSize:  int64(v.FileSize),
			KnownSize: int64(v.KnownSize),
		}
	}

	return &c
}
