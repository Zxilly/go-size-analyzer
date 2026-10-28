package entity

import (
	"cmp"
	"fmt"
	"slices"
)

type FileRange struct {
	Offset uint64 `json:"offset"`
	Size   uint64 `json:"size"`
}

type FileMapping struct {
	Addr uint64
	FileRange
}

type FileRegion struct {
	FileRange
	Class            string         `json:"class"`
	Owners           []string       `json:"owners,omitempty"`
	DefinitionOwners []string       `json:"definition_owners,omitempty"`
	Referrers        []string       `json:"referrers,omitempty"`
	Sources          []string       `json:"sources,omitempty"`
	Evidence         []FileEvidence `json:"evidence,omitempty"`
}

// FileEvidence keeps a claim's package and role attached to its source. A
// source list alone cannot distinguish two packages using different evidence.
type FileEvidence struct {
	Owner    string `json:"owner"`
	Source   string `json:"source"`
	Role     string `json:"role"`
	Boundary string `json:"boundary,omitempty"`
}

type FileCoverage struct {
	Version      int               `json:"version,omitzero"`
	Notes        []string          `json:"notes,omitempty"`
	Attributed   uint64            `json:"attributed"`
	Recognized   uint64            `json:"recognized"`
	Unclassified uint64            `json:"unclassified"`
	Shared       uint64            `json:"shared"`
	BySource     map[string]uint64 `json:"by_source"`
	Regions      []FileRegion      `json:"regions,omitempty"`

	// Projections of the same physical partition used by report totals. They
	// are not additional byte claims and are never serialized independently.
	PackageSizes     map[string]uint64 `json:"-"`
	MetadataSizes    map[string]uint64 `json:"-"`
	AttributedRanges []FileRange       `json:"-"`
}

type FileClaim struct {
	FileRange
	Owner    string
	Source   string
	Role     string
	Boundary string
	Metadata bool
	// 0: unclassified section, 1: recognized structure, 2: attributed bytes.
	Class int
}

type FileLedger struct {
	size          uint64
	claims        []FileClaim
	packageGroups map[string][]string
}

func NewFileLedger(size uint64) *FileLedger { return &FileLedger{size: size} }

// SetPackageGroups supplies each package's containing report nodes, including
// itself. Overlapping claims by a parent and child count once in the parent.
func (l *FileLedger) SetPackageGroups(groups map[string][]string) { l.packageGroups = groups }

func UnionFileSize(ranges []FileRange) uint64 {
	ranges = slices.Clone(ranges)
	slices.SortFunc(ranges, func(a, b FileRange) int { return cmp.Compare(a.Offset, b.Offset) })
	var size, end uint64
	for _, r := range ranges {
		if next := r.Offset + r.Size; next > end {
			size += next - max(end, r.Offset)
			end = next
		}
	}
	return size
}

func (l *FileLedger) Add(c FileClaim) error {
	if c.Class < 0 || c.Class > 2 || c.Offset > l.size || c.Size > l.size-c.Offset {
		return fmt.Errorf("invalid file claim: offset=%d size=%d class=%d file=%d", c.Offset, c.Size, c.Class, l.size)
	}
	if c.Class == 2 {
		if c.Role == "" {
			c.Role = "definition"
		}
		if c.Role != "definition" && c.Role != "reference" {
			return fmt.Errorf("invalid file claim role %q", c.Role)
		}
	}
	if c.Size > 0 {
		l.claims = append(l.claims, c)
	}
	return nil
}

// Finish partitions physical bytes by evidence priority. Ownership is counted
// as a set, so duplicate claims cannot inflate unique or shared coverage.
func (l *FileLedger) Finish(details bool) *FileCoverage {
	type event struct {
		at    uint64
		claim int
		delta int
	}
	events := make([]event, 0, len(l.claims)*2+2)
	for i, c := range l.claims {
		events = append(events, event{c.Offset, i, 1}, event{c.Offset + c.Size, i, -1})
	}
	events = append(events, event{0, -1, 0}, event{l.size, -1, 0})
	slices.SortFunc(events, func(a, b event) int { return cmp.Compare(a.at, b.at) })
	definitions, references, metadata := map[string]int{}, map[string]int{}, map[string]int{}
	evidence := map[FileEvidence]int{}
	sources := [3]map[string]int{{}, {}, {}}
	counts := [3]int{}
	result := &FileCoverage{Version: 2, BySource: map[string]uint64{}, PackageSizes: map[string]uint64{}, MetadataSizes: map[string]uint64{}}
	groups := map[string]struct{}{}
	keys := func(m map[string]int) []string {
		r := make([]string, 0, len(m))
		for k := range m {
			r = append(r, k)
		}
		slices.Sort(r)
		return r
	}
	update := func(m map[string]int, key string, delta int) {
		if key == "" {
			return
		}
		m[key] += delta
		if m[key] == 0 {
			delete(m, key)
		}
	}
	var previous uint64
	for i := 0; i < len(events); {
		at := events[i].at
		if at > previous {
			length := at - previous
			class := 0
			name := "unclassified"
			owners := definitions
			if len(owners) == 0 {
				owners = references
			}
			if counts[2] > 0 {
				class = 2
				name = "attributed"
				result.Attributed += length
				if n := len(result.AttributedRanges); n > 0 && result.AttributedRanges[n-1].Offset+result.AttributedRanges[n-1].Size == previous {
					result.AttributedRanges[n-1].Size += length
				} else {
					result.AttributedRanges = append(result.AttributedRanges, FileRange{Offset: previous, Size: length})
				}
				for source := range sources[2] {
					result.BySource[source] += length
				}
				if len(owners) > 1 {
					result.Shared += length
				}
				clear(groups)
				for owner := range owners {
					if metadata[owner] > 0 {
						result.MetadataSizes[owner] += length
					}
					if ancestors, ok := l.packageGroups[owner]; ok {
						for _, ancestor := range ancestors {
							groups[ancestor] = struct{}{}
						}
					} else {
						groups[owner] = struct{}{}
					}
				}
				for group := range groups {
					result.PackageSizes[group] += length
				}
			} else if counts[1] > 0 {
				class = 1
				name = "recognized"
				result.Recognized += at - previous
			} else {
				result.Unclassified += at - previous
			}
			if details {
				region := FileRegion{FileRange: FileRange{Offset: previous, Size: at - previous}, Class: name, Sources: keys(sources[class])}
				if class == 2 {
					region.Owners = keys(owners)
					region.DefinitionOwners = keys(definitions)
					region.Referrers = keys(references)
					for item := range evidence {
						region.Evidence = append(region.Evidence, item)
					}
					slices.SortFunc(region.Evidence, compareFileEvidence)
				}
				if class == 0 && len(region.Sources) == 0 {
					region.Sources = []string{"unmapped-file-range"}
				}
				if len(result.Regions) > 0 {
					last := &result.Regions[len(result.Regions)-1]
					if last.Class == region.Class && slices.Equal(last.Owners, region.Owners) && slices.Equal(last.Sources, region.Sources) && slices.Equal(last.DefinitionOwners, region.DefinitionOwners) && slices.Equal(last.Referrers, region.Referrers) && slices.Equal(last.Evidence, region.Evidence) {
						last.Size += region.Size
					} else {
						result.Regions = append(result.Regions, region)
					}
				} else {
					result.Regions = append(result.Regions, region)
				}
			}
		}
		for i < len(events) && events[i].at == at {
			e := events[i]
			if e.claim >= 0 {
				c := l.claims[e.claim]
				counts[c.Class] += e.delta
				update(sources[c.Class], c.Source, e.delta)
				if c.Class == 2 {
					if c.Role == "reference" {
						update(references, c.Owner, e.delta)
					} else {
						update(definitions, c.Owner, e.delta)
					}
					if c.Metadata {
						update(metadata, c.Owner, e.delta)
					}
					if details {
						updateFileEvidence(evidence, c, e.delta)
					}
				}
			}
			i++
		}
		previous = at
	}
	return result
}

func updateFileEvidence(evidence map[FileEvidence]int, claim FileClaim, delta int) {
	item := FileEvidence{Owner: claim.Owner, Source: claim.Source, Role: claim.Role, Boundary: claim.Boundary}
	evidence[item] += delta
	if evidence[item] == 0 {
		delete(evidence, item)
	}
}

func compareFileEvidence(a, b FileEvidence) int {
	if n := cmp.Compare(a.Owner, b.Owner); n != 0 {
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
