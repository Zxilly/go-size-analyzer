package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessPackages(t *testing.T) {
	oldPackages := map[string]commonPackage{
		"pkg1": {Size: 100},
		"pkg2": {Size: 200},
	}
	newPackages := map[string]commonPackage{
		"pkg1": {Size: 150},
		"pkg3": {Size: 300},
	}

	expected := []diffPackage{
		{DiffBase{Name: "pkg1", From: 100, To: 150, ChangeType: changeTypeChange}},
		{DiffBase{Name: "pkg3", From: 0, To: 300, ChangeType: changeTypeAdd}},
		{DiffBase{Name: "pkg2", From: 200, To: 0, ChangeType: changeTypeRemove}},
	}

	result := processPackages(newPackages, oldPackages)
	assert.ElementsMatch(t, expected, result)
}

func TestProcessSections(t *testing.T) {
	oldSections := []commonSection{
		{Name: "sec1", FileSize: 100, KnownSize: 50},
		{Name: "sec2", FileSize: 200, KnownSize: 100},
	}
	newSections := []commonSection{
		{Name: "sec1", FileSize: 150, KnownSize: 75},
		{Name: "sec3", FileSize: 300, KnownSize: 150},
	}

	expected := []diffSection{
		{DiffBase: DiffBase{Name: "sec1", From: 50, To: 75, ChangeType: changeTypeChange}, OldFileSize: 100, OldKnownSize: 50, NewFileSize: 150, NewKnownSize: 75},
		{DiffBase: DiffBase{Name: "sec3", From: 0, To: 150, ChangeType: changeTypeAdd}, OldFileSize: 0, OldKnownSize: 0, NewFileSize: 300, NewKnownSize: 150},
		{DiffBase: DiffBase{Name: "sec2", From: 100, To: 0, ChangeType: changeTypeRemove}, OldFileSize: 200, OldKnownSize: 100, NewFileSize: 0, NewKnownSize: 0},
	}

	result := processSections(newSections, oldSections)
	assert.ElementsMatch(t, expected, result)
}

func TestNewDiffResult(t *testing.T) {
	oldResult := &commonResult{
		Packages: map[string]commonPackage{
			"pkg1": {Size: 100},
			"pkg2": {Size: 200},
		},
		Sections: []commonSection{
			{Name: "sec1", FileSize: 100, KnownSize: 50},
			{Name: "sec2", FileSize: 200, KnownSize: 100},
		},
	}
	newResult := &commonResult{
		Packages: map[string]commonPackage{
			"pkg1": {Size: 150},
			"pkg3": {Size: 300},
		},
		Sections: []commonSection{
			{Name: "sec1", FileSize: 150, KnownSize: 75},
			{Name: "sec3", FileSize: 300, KnownSize: 150},
		},
	}

	expected := diffResult{
		Packages: []diffPackage{
			{DiffBase{Name: "pkg3", From: 0, To: 300, ChangeType: changeTypeAdd}},
			{DiffBase{Name: "pkg1", From: 100, To: 150, ChangeType: changeTypeChange}},
			{DiffBase{Name: "pkg2", From: 200, To: 0, ChangeType: changeTypeRemove}},
		},
		Sections: []diffSection{
			{DiffBase: DiffBase{Name: "sec3", From: 0, To: 150, ChangeType: changeTypeAdd}, OldFileSize: 0, OldKnownSize: 0, NewFileSize: 300, NewKnownSize: 150},
			{DiffBase: DiffBase{Name: "sec1", From: 50, To: 75, ChangeType: changeTypeChange}, OldFileSize: 100, OldKnownSize: 50, NewFileSize: 150, NewKnownSize: 75},
			{DiffBase: DiffBase{Name: "sec2", From: 100, To: 0, ChangeType: changeTypeRemove}, OldFileSize: 200, OldKnownSize: 100, NewFileSize: 0, NewKnownSize: 0},
		},
	}

	result := newDiffResult(newResult, oldResult)
	assert.Equal(t, expected, result)
}
