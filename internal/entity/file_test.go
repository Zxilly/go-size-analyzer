package entity_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/global"
	"github.com/stretchr/testify/assert"
	"testing"
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
		assert.NoError(t, err)
		expected := `{"file_path":"/path/to/file","size":60}`
		assert.JSONEq(t, expected, string(data))
	})

	t.Run("HideDetail is false", func(t *testing.T) {
		// Set HideDetail to false
		global.HideDetail = false

		// Call MarshalJSON
		data, err := file.MarshalJSON()

		// Verify the result
		assert.NoError(t, err)
		expected := `
{
	"file_path": "/path/to/file",
	"functions": [{
		"name": "",
		"addr": 0,
		"size": 10,
		"type": "",
		"receiver": "",
		"pcln_size": null
	}, {
		"name": "",
		"addr": 0,
		"size": 20,
		"type": "",
		"receiver": "",
		"pcln_size": null
	}, {
		"name": "",
		"addr": 0,
		"size": 30,
		"type": "",
		"receiver": "",
		"pcln_size": null
	}]
}`
		assert.JSONEq(t, expected, string(data))
	})
}
