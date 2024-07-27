package knowninfo

import (
	"log/slog"
)

func (k *KnownInfo) LoadSectionMap() error {
	slog.Info("Loading sections...")

	store := k.Wrapper.LoadSections()
	store.BuildCache()

	slog.Info("Loaded sections")

	k.Sects = store
	return k.Sects.AssertSize(k.Size)
}
