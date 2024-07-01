package result

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type Result struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`

	Analyzers []entity.Analyzer `json:"analyzers"`
	Packages  entity.PackageMap `json:"packages"`
	Sections  []*entity.Section `json:"sections"`
}
