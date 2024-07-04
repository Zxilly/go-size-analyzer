package diff

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func requireAnalyzeModeSame(oldResult, newResult *commonResult) bool {
	oldModes := utils.NewSet[entity.Analyzer]()
	for _, v := range oldResult.Analyzers {
		oldModes.Add(v)
	}

	newModes := utils.NewSet[entity.Analyzer]()
	for _, v := range newResult.Analyzers {
		newModes.Add(v)
	}

	return oldModes.Equals(newModes)
}
