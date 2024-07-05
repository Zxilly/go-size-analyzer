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
		{diffBase{Name: "pkg1", From: 100, To: 150, ChangeType: changeTypeChange}},
		{diffBase{Name: "pkg3", From: 0, To: 300, ChangeType: changeTypeAdd}},
		{diffBase{Name: "pkg2", From: 200, To: 0, ChangeType: changeTypeRemove}},
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
		{diffBase: diffBase{Name: "sec1", From: 50, To: 75, ChangeType: changeTypeChange}, oldFileSize: 100, oldKnownSize: 50, newFileSize: 150, newKnownSize: 75},
		{diffBase: diffBase{Name: "sec3", From: 0, To: 150, ChangeType: changeTypeAdd}, oldFileSize: 0, oldKnownSize: 0, newFileSize: 300, newKnownSize: 150},
		{diffBase: diffBase{Name: "sec2", From: 100, To: 0, ChangeType: changeTypeRemove}, oldFileSize: 200, oldKnownSize: 100, newFileSize: 0, newKnownSize: 0},
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
			{diffBase{Name: "pkg3", From: 0, To: 300, ChangeType: changeTypeAdd}},
			{diffBase{Name: "pkg1", From: 100, To: 150, ChangeType: changeTypeChange}},
			{diffBase{Name: "pkg2", From: 200, To: 0, ChangeType: changeTypeRemove}},
		},
		Sections: []diffSection{
			{diffBase: diffBase{Name: "sec3", From: 0, To: 150, ChangeType: changeTypeAdd}, oldFileSize: 0, oldKnownSize: 0, newFileSize: 300, newKnownSize: 150},
			{diffBase: diffBase{Name: "sec1", From: 50, To: 75, ChangeType: changeTypeChange}, oldFileSize: 100, oldKnownSize: 50, newFileSize: 150, newKnownSize: 75},
			{diffBase: diffBase{Name: "sec2", From: 100, To: 0, ChangeType: changeTypeRemove}, oldFileSize: 200, oldKnownSize: 100, newFileSize: 0, newKnownSize: 0},
		},
	}

	result := newDiffResult(newResult, oldResult)
	assert.Equal(t, expected, result)
}
