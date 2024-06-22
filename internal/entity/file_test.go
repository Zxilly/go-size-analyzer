package entity_test

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
)

func TestFile_MarshalJSON(t *testing.T) {
	// Create a sample File instance
	file := &entity.File{
		FilePath: "/path/to/file",
		Functions: []*entity.Function{
			{CodeSize: 10},
			{CodeSize: 20},
			{CodeSize: 30},
		},
	}

	t.Run("Compact mode", func(t *testing.T) {
		data, err := json.Marshal(file, json.WithMarshalers(marshaler.GetFileCompactMarshaler()))
		// Verify the result
		require.NoError(t, err)
		expected := `{"file_path":"/path/to/file","pcln_size":0,"size":60}`
		assert.JSONEq(t, expected, string(data))
	})

	t.Run("Full mode", func(t *testing.T) {
		data, err := json.Marshal(file)

		// Verify the result
		require.NoError(t, err)
		expected := `
{
    "file_path": "/path/to/file",
    "functions": [
        {
            "name": "",
            "addr": 0,
            "code_size": 10,
            "type": "",
            "receiver": "",
            "pcln_size": {
                "name": 0,
                "pcfile": 0,
                "pcsp": 0,
                "pcln": 0,
                "header": 0,
                "funcdata": 0,
                "pcdata": {}
            }
        },
        {
            "name": "",
            "addr": 0,
            "code_size": 20,
            "type": "",
            "receiver": "",
            "pcln_size": {
                "name": 0,
                "pcfile": 0,
                "pcsp": 0,
                "pcln": 0,
                "header": 0,
                "funcdata": 0,
                "pcdata": {}
            }
        },
        {
            "name": "",
            "addr": 0,
            "code_size": 30,
            "type": "",
            "receiver": "",
            "pcln_size": {
                "name": 0,
                "pcfile": 0,
                "pcsp": 0,
                "pcln": 0,
                "header": 0,
                "funcdata": 0,
                "pcdata": {}
            }
        }
    ]
}`
		assert.JSONEq(t, expected, string(data))
	})
}
