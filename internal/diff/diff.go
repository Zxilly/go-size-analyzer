package diff

import (
	"cmp"
	"slices"
)

type diffResult struct {
	Size     int64         `json:"size"`
	Packages []diffPackage `json:"packages"`
	Sections []diffSection `json:"sections"`
}

type changeType string

const (
	changeTypeAdd    changeType = "add"
	changeTypeRemove changeType = "remove"
	changeTypeChange changeType = "change"
)

type diffBase struct {
	Name       string     `json:"name"`
	From       int64      `json:"from"`
	To         int64      `json:"to"`
	ChangeType changeType `json:"change_type"`
}

func diffBaseCmp(a, b diffBase) int {
	return -cmp.Compare(a.To-a.From, b.To-b.From)
}

type diffPackage struct {
	diffBase
}

type diffSection struct {
	diffBase
	oldFileSize  int64
	oldKnownSize int64
	newFileSize  int64
	newKnownSize int64
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
			diffBase: diffBase{Name: k, From: fromSize, To: v.Size, ChangeType: typ},
		})
	}

	for k, v := range oldPackages {
		if _, ok := newPackages[k]; !ok {
			ret = append(ret, diffPackage{
				diffBase: diffBase{Name: k, From: v.Size, To: 0, ChangeType: changeTypeRemove},
			})
		}
	}

	return
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
			diffBase:     diffBase{Name: k, From: fromSize, To: v.UnknownSize(), ChangeType: typ},
			oldFileSize:  fromFileSize,
			oldKnownSize: fromKnownSize,
			newFileSize:  v.FileSize,
			newKnownSize: v.KnownSize,
		})
	}

	for k, v := range oldSectionsMap {
		if _, ok := newSectionsMap[k]; !ok {
			ret = append(ret, diffSection{
				diffBase:     diffBase{Name: k, From: v.UnknownSize(), To: 0, ChangeType: changeTypeRemove},
				oldFileSize:  v.FileSize,
				oldKnownSize: v.KnownSize,
			})
		}
	}

	return
}

func newDiffResult(newResult, oldResult *commonResult) diffResult {
	ret := diffResult{
		Packages: processPackages(newResult.Packages, oldResult.Packages),
		Sections: processSections(newResult.Sections, oldResult.Sections),
	}

	slices.SortFunc(ret.Packages, func(a, b diffPackage) int {
		return diffBaseCmp(a.diffBase, b.diffBase)
	})
	slices.SortFunc(ret.Sections, func(a, b diffSection) int {
		return diffBaseCmp(a.diffBase, b.diffBase)
	})

	return ret
}
