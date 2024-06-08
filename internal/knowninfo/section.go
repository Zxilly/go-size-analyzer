package knowninfo

import (
	"github.com/Zxilly/go-size-analyzer/internal/section"
	"log/slog"
)

func (k *KnownInfo) LoadSectionMap() error {
	slog.Info("Loading sections...")

	sections := k.Wrapper.LoadSections()

	slog.Info("Loading sections done")

	k.Sects = &section.SectionMap{
		Sections: sections,
	}
	return k.Sects.AssertSize(k.Size)
}
