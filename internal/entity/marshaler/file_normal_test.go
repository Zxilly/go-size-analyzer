package marshaler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFileDetailsPreserveTotalsWithoutChangingCompact(t *testing.T) {
	zero := uint64(0)
	f := entity.File{FilePath: "source.go", Functions: []*entity.Function{{Name: "Read", FullName: "example.Read", CodeSize: 64, PclnFileSize: &zero, ReferencedBytes: &zero}}}
	for _, tc := range []struct {
		name      string
		marshaler *json.Marshalers
		details   bool
	}{
		{"compact", GetFileCompactMarshaler(), false},
		{"detail", GetFileDetailMarshaler(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(f, json.WithMarshalers(tc.marshaler))
			require.NoError(t, err)
			var value map[string]jsontext.Value
			require.NoError(t, json.Unmarshal(encoded, &value))
			require.JSONEq(t, "64", string(value["size"]))
			require.JSONEq(t, "0", string(value["pcln_size"]))
			if tc.details {
				var functions []map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(value["functions"], &functions))
				require.Len(t, functions, 1)
				require.JSONEq(t, "0", string(functions[0]["referenced_bytes"]))
				require.JSONEq(t, `"example.Read"`, string(functions[0]["full_name"]))
			} else {
				require.NotContains(t, value, "functions")
			}
		})
	}
}
