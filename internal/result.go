package internal

import (
	"golang.org/x/exp/maps"
)

type ResultSection struct {
	Name      string `json:"name"`
	KnownSize uint64 `json:"known_size"`
	Size      uint64 `json:"size"`
}

type Result struct {
	Name     string     `json:"name"`
	Size     uint64     `json:"size"`
	Packages PackageMap `json:"packages"`
	Sections []*Section `json:"sections"`
}

func BuildResult(name string, k *KnownInfo) *Result {
	r := &Result{
		Name:     name,
		Size:     k.Size,
		Packages: k.Packages.topPkgs,
		Sections: maps.Values(k.SectionMap.Sections),
	}
	return r
}
