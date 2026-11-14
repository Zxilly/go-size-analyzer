package entity

import (
	"cmp"
	"slices"
)

// FunctionEvidence describes a detected reference to physical bytes. It does
// not claim exclusive ownership or imply that removing the function saves them.
type FunctionEvidence struct {
	FileRange
	Section  string `json:"section"`
	Source   string `json:"source"`
	Role     string `json:"role"`
	Boundary string `json:"boundary"`
}

const functionEvidenceLimit = 8

// SetReferenceSummary projects the existing instruction evidence onto the same
// native, file-backed sections used by the byte ledger. Keep this separate from
// Size: references may overlap definitions and references from other functions.
func (f *Function) SetReferenceSummary(sections []*Section, details, available bool) {
	f.ReferencedBytes, f.ReferenceEvidence, f.ReferenceEvidenceCount = nil, nil, 0
	if !available || f.Wasm {
		return
	}
	var ranges []FileRange
	var evidence []FunctionEvidence
	for addr := range f.disasm.All {
		if addr.Role != AddrRoleReference || addr.Size == 0 || addr.Size > ^uint64(0)-addr.Addr {
			continue
		}
		for _, s := range sections {
			if s.OnlyInMemory || s.VirtualSection || s.Debug || s.ContentType == SectionContentOther || s.FileSize == 0 {
				continue
			}
			length := min(s.Size, s.FileSize)
			if length > ^uint64(0)-s.Addr || length > ^uint64(0)-s.Offset {
				continue
			}
			lo, hi := max(addr.Addr, s.Addr), min(addr.Addr+addr.Size, s.Addr+length)
			if lo >= hi {
				continue
			}
			r := FileRange{Offset: s.Offset + lo - s.Addr, Size: hi - lo}
			ranges = append(ranges, r)
			if details {
				evidence = append(evidence, FunctionEvidence{FileRange: r, Section: s.Name, Source: addr.SourceType, Role: addr.Role, Boundary: addr.Boundary})
			}
		}
	}
	n := UnionFileSize(ranges)
	f.ReferencedBytes = &n
	if !details {
		return
	}
	// Sorting also makes the bounded sample independent of map iteration order.
	slices.SortFunc(evidence, compareFunctionEvidence)
	evidence = slices.Compact(evidence)
	f.ReferenceEvidenceCount = len(evidence)
	if len(evidence) > functionEvidenceLimit {
		evidence = evidence[:functionEvidenceLimit]
	}
	f.ReferenceEvidence = slices.Clone(evidence)
}

func compareFunctionEvidence(a, b FunctionEvidence) int {
	if n := cmp.Compare(a.Offset, b.Offset); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Size, b.Size); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Section, b.Section); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Source, b.Source); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Role, b.Role); n != 0 {
		return n
	}
	return cmp.Compare(a.Boundary, b.Boundary)
}
