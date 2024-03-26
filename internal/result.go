package internal

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"golang.org/x/exp/maps"
	"path"
)

type ResultSection struct {
	Name      string `json:"name"`
	KnownSize uint64 `json:"known_size"`
	Size      uint64 `json:"size"`
}

type Result struct {
	Name     string            `json:"name"`
	Size     uint64            `json:"size"`
	Packages entity.PackageMap `json:"packages"`
	Sections []*entity.Section `json:"sections"`
}

func BuildResult(name string, k *KnownInfo) *Result {
	r := &Result{
		Name:     path.Base(name),
		Size:     k.Size,
		Packages: k.Deps.topPkgs,
		Sections: maps.Values(k.Sects.Sections),
	}
	return r
}
