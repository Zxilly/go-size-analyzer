package diff

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func TestDiffJSONAndBinary(t *testing.T) {
	tests := []struct {
		name   string
		old    string
		new    string
		format string
	}{
		{
			name:   "json to binary",
			old:    test.GetTestJSONPath(t),
			new:    test.GetTestDiffBinPath(t),
			format: "json",
		},
		{
			name:   "binary to binary",
			old:    test.GetTestBinPath(t),
			new:    test.GetTestDiffBinPath(t),
			format: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			require.NoError(t, Diff(io.Discard, DOptions{
				OldTarget: tt.old,
				NewTarget: tt.new,
				Format:    tt.format,
			}))
		})
	}
}
