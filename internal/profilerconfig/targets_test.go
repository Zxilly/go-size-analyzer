package profilerconfig

import (
	"testing"

	"github.com/knadh/profiler"
	"github.com/stretchr/testify/require"
)

func TestTargets(t *testing.T) {
	t.Run("windows", func(t *testing.T) {
		require.Equal(t,
			[]int{profiler.Cpu, profiler.Goroutine, profiler.Block, profiler.ThreadCreate, profiler.Trace},
			Targets("windows"),
		)
	})

	t.Run("other", func(t *testing.T) {
		require.Equal(t,
			[]int{profiler.Cpu, profiler.Mutex, profiler.Goroutine, profiler.Block, profiler.ThreadCreate, profiler.Trace},
			Targets("linux"),
		)
	})
}
