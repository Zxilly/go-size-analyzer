package knowninfo

import (
	"log/slog"

	"github.com/Zxilly/go-size-analyzer/internal/section"
)

func (k *KnownInfo) LoadSectionMap() error {
	slog.Info("Loading sections...")

	sections := k.Wrapper.LoadSections()

	slog.Info("Loading sections done")

	k.Sects = &section.Store{
		Sections: sections,
	}
	return k.Sects.AssertSize(k.Size)
}
