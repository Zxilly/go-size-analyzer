package diff

import (
	"bytes"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func TestCommonResultFromFullResult(t *testing.T) {
	fullResult := test.GetTestResult(t)

	cr := fromResult(fullResult)

	jsonData := new(bytes.Buffer)

	err := printer.JSON(fullResult, jsonData, &printer.JSONOption{
		Indent:     nil,
		HideDetail: false,
	})
	require.NoError(t, err)

	crFromJSON := new(commonResult)
	err = json.UnmarshalRead(jsonData, crFromJSON)
	require.NoError(t, err)

	assert.Equal(t, cr, crFromJSON) //nolint:testifylint // workaround github.com/Antonboom/testifylint/issues/198
}

func TestCommonResultFromFullAndCompactJSON(t *testing.T) {
	fullResult := test.GetTestResult(t)

	compactJSONData := new(bytes.Buffer)
	err := printer.JSON(fullResult, compactJSONData, &printer.JSONOption{
		Indent:     nil,
		HideDetail: true,
	})
	require.NoError(t, err)

	fullJSONData := new(bytes.Buffer)
	err = printer.JSON(fullResult, fullJSONData, &printer.JSONOption{
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

	assert.Equal(t, crFromCompactJSON.Size, crFromFullJSON.Size)
	assert.Equal(t, crFromCompactJSON.Sections, crFromFullJSON.Sections)
	comparison := newDiffResult(crFromCompactJSON, crFromFullJSON)
	assert.Empty(t, comparison.Packages)
	assert.Empty(t, comparison.Functions, "omitted compact details do not mean functions were deleted")
	assert.Empty(t, comparison.Symbols, "compact retains the same symbol inventory")
	assert.Contains(t, diagnosticCodes(comparison.Diagnostics), "function_details_unavailable")
}
