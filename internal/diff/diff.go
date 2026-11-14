package diff

import (
	"cmp"
	"runtime/debug"
	"slices"
)

type diffResult struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`

	OldSize int64 `json:"old_size"`
	NewSize int64 `json:"new_size"`

	Packages     []diffPackage    `json:"packages"`
	Sections     []diffSection    `json:"sections"`
	Functions    []diffFunction   `json:"functions,omitempty"`
	Symbols      []diffSymbol     `json:"symbols,omitempty"`
	Diagnostics  []diagnostic     `json:"diagnostics,omitempty"`
	OldBuildInfo *debug.BuildInfo `json:"old_build_info,omitempty"`
	NewBuildInfo *debug.BuildInfo `json:"new_build_info,omitempty"`
	BuildChanges []buildChange    `json:"build_changes,omitempty"`
}

type changeType string

const (
	changeTypeAdd    changeType = "add"
	changeTypeRemove changeType = "remove"
	changeTypeChange changeType = "change"
)

type Base struct {
	Name       string     `json:"name"`
	From       int64      `json:"from"`
	To         int64      `json:"to"`
	ChangeType changeType `json:"change_type"`
}

func diffBaseCmp(a, b Base) int {
	return -cmp.Compare(a.To-a.From, b.To-b.From)
}

type diffPackage struct {
	Base
}

type diffSection struct {
	Base
	OldFileSize  int64 `json:"old_file_size"`
	OldKnownSize int64 `json:"old_known_size"`
	NewFileSize  int64 `json:"new_file_size"`
	NewKnownSize int64 `json:"new_known_size"`
}

func processPackages(newPackages, oldPackages map[string]commonPackage) (ret []diffPackage) {
	for k, v := range newPackages {
		typ := changeTypeAdd
		fromSize := int64(0)
		if oldV, ok := oldPackages[k]; ok {
			if v.Size == oldV.Size {
				continue
			}
			typ = changeTypeChange
			fromSize = oldV.Size
		}
		ret = append(ret, diffPackage{
			Base: Base{Name: k, From: fromSize, To: v.Size, ChangeType: typ},
		})
	}

	for k, v := range oldPackages {
		if _, ok := newPackages[k]; !ok {
			ret = append(ret, diffPackage{
				Base: Base{Name: k, From: v.Size, To: 0, ChangeType: changeTypeRemove},
			})
		}
	}

	return ret
}

func processSections(newSections, oldSections []commonSection) (ret []diffSection) {
	newSectionsMap := make(map[string]commonSection)
	oldSectionsMap := make(map[string]commonSection)

	for _, v := range newSections {
		newSectionsMap[v.Name] = v
	}
	for _, v := range oldSections {
		oldSectionsMap[v.Name] = v
	}

	for k, v := range newSectionsMap {
		typ := changeTypeAdd
		var fromSize, fromFileSize, fromKnownSize int64

		if oldV, ok := oldSectionsMap[k]; ok {
			if v.UnknownSize() == oldV.UnknownSize() {
				continue
			}
			typ = changeTypeChange
			fromSize = oldV.UnknownSize()
			fromFileSize = oldV.FileSize
			fromKnownSize = oldV.KnownSize
		}

		ret = append(ret, diffSection{
			Base:         Base{Name: k, From: fromSize, To: v.UnknownSize(), ChangeType: typ},
			OldFileSize:  fromFileSize,
			OldKnownSize: fromKnownSize,
			NewFileSize:  v.FileSize,
			NewKnownSize: v.KnownSize,
		})
	}

	for k, v := range oldSectionsMap {
		if _, ok := newSectionsMap[k]; !ok {
			ret = append(ret, diffSection{
				Base:         Base{Name: k, From: v.UnknownSize(), To: 0, ChangeType: changeTypeRemove},
				OldFileSize:  v.FileSize,
				OldKnownSize: v.KnownSize,
			})
		}
	}

	return ret
}

func newDiffResult(newResult, oldResult *commonResult) diffResult {
	ret := diffResult{
		OldName: oldResult.Name,
		NewName: newResult.Name,
		OldSize: oldResult.Size,
		NewSize: newResult.Size,

		Packages: processPackages(newResult.Packages, oldResult.Packages),
		Sections: processSections(newResult.Sections, oldResult.Sections),
	}
	ret.Functions, ret.Diagnostics = processFunctions(newResult, oldResult)
	symbols, symbolNotes := processSymbols(newResult, oldResult)
	ret.Symbols = symbols
	ret.Diagnostics = append(ret.Diagnostics, symbolNotes...)
	ret.OldBuildInfo, ret.NewBuildInfo = oldResult.BuildInfo, newResult.BuildInfo
	ret.BuildChanges, ret.Diagnostics = compareContext(oldResult, newResult, ret.Diagnostics)
	slices.SortFunc(ret.Diagnostics, func(a, b diagnostic) int {
		if n := cmp.Compare(a.Code, b.Code); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Package, b.Package); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Identity, b.Identity); n != 0 {
			return n
		}
		return cmp.Compare(a.Message, b.Message)
	})

	slices.SortFunc(ret.Packages, func(a, b diffPackage) int {
		return diffBaseCmp(a.Base, b.Base)
	})
	slices.SortFunc(ret.Sections, func(a, b diffSection) int {
		return diffBaseCmp(a.Base, b.Base)
	})

	return ret
}
