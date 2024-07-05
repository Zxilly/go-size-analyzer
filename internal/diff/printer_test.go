package diff

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffStringChangeTypeChangeReturnsPercentage(t *testing.T) {
	b := DiffBase{From: 100, To: 150, ChangeType: changeTypeChange}
	assert.Equal(t, "+50.00%", diffString(b))
	b = DiffBase{From: 150, To: 100, ChangeType: changeTypeChange}
	assert.Equal(t, "-33.33%", diffString(b))
}

func TestDiffStringChangeTypeAddReturnsAdd(t *testing.T) {
	b := DiffBase{ChangeType: changeTypeAdd}
	assert.Equal(t, "add", diffString(b))
}

func TestDiffStringChangeTypeRemoveReturnsRemove(t *testing.T) {
	b := DiffBase{ChangeType: changeTypeRemove}
	assert.Equal(t, "remove", diffString(b))
}

func TestSignedBytesStringPositiveValueReturnsPlusSign(t *testing.T) {
	assert.Equal(t, "+1 B", signedBytesString(1))
}

func TestSignedBytesStringNegativeValueReturnsMinusSign(t *testing.T) {
	assert.Equal(t, "-1 B", signedBytesString(-1))
}

func TestTextRendersTableCorrectly(t *testing.T) {
	var buf bytes.Buffer
	r := &diffResult{
		OldName: "old",
		NewName: "new",
		OldSize: 100,
		NewSize: 150,
		Sections: []diffSection{
			{DiffBase: DiffBase{Name: "sec1", From: 100, To: 150, ChangeType: changeTypeChange}},
		},
		Packages: []diffPackage{
			{DiffBase: DiffBase{Name: "pkg1", From: 100, To: 150, ChangeType: changeTypeChange}},
		},
	}
	err := text(r, &buf)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Diff between old and new")
	assert.Contains(t, buf.String(), "sec1")
	assert.Contains(t, buf.String(), "pkg1")
}

func TestTextHandlesEmptyResultWithoutError(t *testing.T) {
	var buf bytes.Buffer
	r := &diffResult{}
	err := text(r, &buf)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Diff between  and ")
}
