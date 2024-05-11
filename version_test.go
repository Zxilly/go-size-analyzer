package gsv

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

func TestGetStaticVersion(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetStaticVersion",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetStaticVersion(), "GetStaticVersion()")
		})
	}
}
