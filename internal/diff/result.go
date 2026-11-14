package diff

import (
	"runtime/debug"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

// commonResult keeps these json fields are strict subset of result.Result
type commonResult struct {
	Name string `json:"name"`
	Size int64  `json:"size"`

	Analyzers []entity.Analyzer        `json:"analyzers"`
	Packages  map[string]commonPackage `json:"packages"`
	Sections  []commonSection          `json:"sections"`
	BuildInfo *debug.BuildInfo         `json:"build_info,omitempty"`
	Coverage  *commonCoverage          `json:"coverage,omitempty"`
}

type commonPackage struct {
	Name        string                   `json:"name"`
	Size        int64                    `json:"size"`
	SubPackages map[string]commonPackage `json:"subPackages"`
	Files       *[]commonFile            `json:"files"`
	Symbols     *[]commonSymbol          `json:"symbols"`
}

type commonSymbol struct {
	Name string  `json:"name"`
	Type string  `json:"type"`
	Addr *uint64 `json:"addr,omitempty"`
	Size *uint64 `json:"size,omitempty"`
}

type commonFile struct {
	Functions *[]commonFunction `json:"functions"`
}

// Pointers distinguish a measured zero from a field absent in older/compact
// reports. Missing metrics and function lists must never imply deletion.
type commonFunction struct {
	Name            string  `json:"name"`
	FullName        string  `json:"full_name,omitempty"`
	Type            string  `json:"type"`
	Receiver        string  `json:"receiver"`
	CodeSize        *uint64 `json:"code_size,omitempty"`
	PclnBytes       *uint64 `json:"pcln_bytes,omitempty"`
	ReferencedBytes *uint64 `json:"referenced_bytes,omitempty"`
}

type commonCoverage struct {
	Version int `json:"version"`
}

type commonSection struct {
	Name string `json:"name"`

	FileSize  int64 `json:"file_size"`
	KnownSize int64 `json:"known_size"`
}

func (c commonSection) UnknownSize() int64 {
	return c.FileSize - c.KnownSize
}

func fromResult(r *result.Result) *commonResult {
	c := commonResult{
		Name:      r.Name,
		Size:      int64(r.Size),
		Analyzers: r.Analyzers,
		Packages:  make(map[string]commonPackage),
		Sections:  make([]commonSection, len(r.Sections)),
		BuildInfo: r.BuildInfo,
	}
	if r.Coverage != nil {
		c.Coverage = &commonCoverage{Version: r.Coverage.Version}
	}

	for k, v := range r.Packages {
		c.Packages[k] = fromPackage(v)
	}

	for i, v := range r.Sections {
		c.Sections[i] = commonSection{
			Name:      v.Name,
			FileSize:  int64(v.FileSize),
			KnownSize: int64(v.KnownSize),
		}
	}

	return &c
}

func fromPackage(p *entity.Package) commonPackage {
	files := make([]commonFile, len(p.Files))
	symbols := make([]commonSymbol, len(p.Symbols))
	c := commonPackage{Name: p.Name, Size: int64(p.Size), Files: &files, Symbols: &symbols, SubPackages: make(map[string]commonPackage, len(p.SubPackages))}
	for i, s := range p.Symbols {
		addr, size := s.Addr, s.Size
		symbols[i] = commonSymbol{Name: s.Name, Type: s.Type, Addr: &addr, Size: &size}
	}
	for i, file := range p.Files {
		functions := make([]commonFunction, len(file.Functions))
		for j, fn := range file.Functions {
			codeSize := fn.CodeSize
			functions[j] = commonFunction{Name: fn.Name, FullName: fn.FullName, Type: fn.Type, Receiver: fn.Receiver, CodeSize: &codeSize, PclnBytes: fn.PclnFileSize, ReferencedBytes: fn.ReferencedBytes}
		}
		files[i].Functions = &functions
	}
	for key, child := range p.SubPackages {
		c.SubPackages[key] = fromPackage(child)
	}
	return c
}
