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

func newDiffResult(newResult, oldResult *commonResult) diffResult {
	ret := diffResult{
		Packages: make([]diffPackage, 0),
		Sections: make([]diffSection, 0),
	}

	// diff packages
	for k, v := range newResult.Packages {
		if oldV, ok := oldResult.Packages[k]; ok {
			if v.Size != oldV.Size {
				ret.Packages = append(ret.Packages, diffPackage{
					diffBase: diffBase{
						Name:       k,
						From:       oldV.Size,
						To:         v.Size,
						ChangeType: changeTypeChange,
					},
				})
			}
		} else {
			ret.Packages = append(ret.Packages, diffPackage{
				diffBase: diffBase{
					Name:       k,
					From:       0,
					To:         v.Size,
					ChangeType: changeTypeAdd,
				},
			})
		}
	}

	for k, v := range oldResult.Packages {
		if _, ok := newResult.Packages[k]; !ok {
			ret.Packages = append(ret.Packages, diffPackage{
				diffBase: diffBase{
					Name:       k,
					From:       v.Size,
					To:         0,
					ChangeType: changeTypeRemove,
				},
			})
		}
	}

	// diff sections
	newSections := make(map[string]commonSection)
	oldSections := make(map[string]commonSection)

	for _, v := range newResult.Sections {
		newSections[v.Name] = v
	}
	for _, v := range oldResult.Sections {
		oldSections[v.Name] = v
	}

	for k, v := range newSections {
		if oldV, ok := oldSections[k]; ok {
			if v.UnknownSize() != oldV.UnknownSize() {
				ret.Sections = append(ret.Sections, diffSection{
					diffBase: diffBase{
						Name:       k,
						From:       oldV.FileSize,
						To:         v.FileSize,
						ChangeType: changeTypeChange,
					},
					oldFileSize:  oldV.FileSize,
					oldKnownSize: oldV.KnownSize,
					newFileSize:  v.FileSize,
					newKnownSize: v.KnownSize,
				})
			}
		} else {
			ret.Sections = append(ret.Sections, diffSection{
				diffBase: diffBase{
					Name:       k,
					From:       0,
					To:         v.FileSize,
					ChangeType: changeTypeAdd,
				},
				newFileSize:  v.FileSize,
				newKnownSize: v.KnownSize,
			})
		}
	}

	for k, v := range oldSections {
		if _, ok := newSections[k]; !ok {
			ret.Sections = append(ret.Sections, diffSection{
				diffBase: diffBase{
					Name:       k,
					From:       v.FileSize,
					To:         0,
					ChangeType: changeTypeRemove,
				},
				oldFileSize:  v.FileSize,
				oldKnownSize: v.KnownSize,
			})
		}
	}

	slices.SortFunc(ret.Packages, func(a, b diffPackage) int {
		return diffBaseCmp(a.diffBase, b.diffBase)
	})
	slices.SortFunc(ret.Sections, func(a, b diffSection) int {
		return diffBaseCmp(a.diffBase, b.diffBase)
	})

	return ret
}
