package entity_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestOptionalNumericMetadataFieldsJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		key   string
		want  float64
	}{
		{"zero package metadata", &entity.Package{}, "metadata_size", 0},
		{"nonzero package metadata", &entity.Package{MetadataSize: 64}, "metadata_size", 64},
		{"legacy coverage", &entity.FileCoverage{}, "version", 0},
		{"current coverage", &entity.FileCoverage{Version: 2}, "version", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.value)
			require.NoError(t, err)
			var object map[string]any
			require.NoError(t, json.Unmarshal(data, &object))
			if tc.want == 0 {
				require.NotContains(t, object, tc.key)
			} else {
				require.InDelta(t, tc.want, object[tc.key], 0)
			}
		})
	}
}
