package result

import (
	"runtime/debug"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type Result struct {
	Name      string           `json:"name"`
	Size      uint64           `json:"size"`
	BuildInfo *debug.BuildInfo `json:"build_info,omitempty"`

	Analyzers []entity.Analyzer    `json:"analyzers"`
	Packages  entity.PackageMap    `json:"packages"`
	Sections  []*entity.Section    `json:"sections"`
	Coverage  *entity.FileCoverage `json:"coverage,omitempty"`
}
