package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/global"
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

	t.Run("HideDetail is true", func(t *testing.T) {
		// Set HideDetail to true
		global.HideDetail = true

		// Call MarshalJSON
		data, err := file.MarshalJSON()

		// Verify the result
		require.NoError(t, err)
		expected := `{"file_path":"/path/to/file","pcln_size":0,"size":60}`
		assert.JSONEq(t, expected, string(data))
	})

	t.Run("HideDetail is false", func(t *testing.T) {
		// Set HideDetail to false
		global.HideDetail = false

		// Call MarshalJSON
		data, err := file.MarshalJSON()

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
                "pcdata": null
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
                "pcdata": null
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
                "pcdata": null
            }
        }
    ]
}`
		assert.JSONEq(t, expected, string(data))
	})
}
