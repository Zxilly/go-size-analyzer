package gsa

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSprintVersion(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		got := SprintVersion()

		keys := []string{"Version", "Go Version", "Platform"}
		for _, key := range keys {
			assert.Contains(t, got, key)
		}
	})

	t.Run("Fake release", func(t *testing.T) {
		buildDate = time.Now().Format(time.RFC3339)
		commitDate = time.Now().Format(time.RFC3339)
		dirtyBuild = "true"

		got := SprintVersion()

		keys := []string{"Version", "Go Version", "Platform", "Build Date", "Commit Date", "Dirty Build"}
		for _, key := range keys {
			assert.Contains(t, got, key)
		}
	})
}
