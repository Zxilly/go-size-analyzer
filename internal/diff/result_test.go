package diff

import (
	"bytes"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/test"
	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommonResultFromFullResult(t *testing.T) {
	fullResult := test.GetTestResult(t)

	cr := fromResult(fullResult)

	jsonData := new(bytes.Buffer)

	err := printer.JSON(fullResult, &printer.JSONOption{
		Writer:     jsonData,
		Indent:     nil,
		HideDetail: false,
	})
	require.NoError(t, err)

	crFromJson := new(commonResult)
	err = json.UnmarshalRead(jsonData, crFromJson)
	require.NoError(t, err)

	assert.Equal(t, cr, crFromJson)
}

func TestCommonResultFromFullAndCompactJSON(t *testing.T) {
	fullResult := test.GetTestResult(t)

	compactJSONData := new(bytes.Buffer)
	err := printer.JSON(fullResult, &printer.JSONOption{
		Writer:     compactJSONData,
		Indent:     nil,
		HideDetail: true,
	})
	require.NoError(t, err)

	fullJSONData := new(bytes.Buffer)
	err = printer.JSON(fullResult, &printer.JSONOption{
		Writer:     fullJSONData,
		Indent:     nil,
		HideDetail: false,
	})
	require.NoError(t, err)

	crFromCompactJSON := new(commonResult)
	crFromFullJSON := new(commonResult)

	err = json.UnmarshalRead(compactJSONData, crFromCompactJSON)
	require.NoError(t, err)

	err = json.UnmarshalRead(fullJSONData, crFromFullJSON)
	require.NoError(t, err)

	assert.Equal(t, crFromCompactJSON, crFromFullJSON)
}
