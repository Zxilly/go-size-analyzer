package knowninfo

import (
	"errors"
	"log/slog"
)

func (k *KnownInfo) LoadSectionMap() error {
	slog.Info("Loading sections...")

	store := k.Wrapper.LoadSections()
	if store == nil {
		return errors.New("failed to load sections")
	}

	store.BuildCache()

	slog.Info("Loaded sections")

	k.Sects = store
	return k.Sects.AssertSize(k.Size)
}
