package gsa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSprintVersion(t *testing.T) {
	got := SprintVersion()

	keys := []string{"Version", "Git Commit", "Build Date", "Commit Date", "Dirty Build", "Go Version", "Platform"}
	for _, key := range keys {
		assert.Contains(t, got, key)
	}
}
