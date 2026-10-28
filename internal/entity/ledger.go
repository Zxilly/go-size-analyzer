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
	events        []fileEvent
	attributes    []fileAttributes
	attributeIDs  map[fileAttributeKey]int
	lastAttribute int
	names         []string
	nameIDs       map[string]int
	packageGroups map[string][]string
}

// Claims commonly repeat the same provenance across thousands of ranges.
// Retain that provenance once and keep only a signed ID at each endpoint.
// The event storage has no pointers for the garbage collector to scan.
type fileEvent struct {
	at        uint64
	attribute int // positive: start, negative: end; zero: file boundary
}

type fileAttributeKey struct {
	FileEvidence
	class    int
	metadata bool
}

type fileAttributes struct {
	fileAttributeKey
	owner, source int
}

func NewFileLedger(size uint64) *FileLedger {
	return &FileLedger{
		size: size, events: []fileEvent{{at: 0}, {at: size}},
		attributeIDs: make(map[fileAttributeKey]int),
		names:        []string{""}, nameIDs: map[string]int{"": 0},
	}
}

func (l *FileLedger) nameID(name string) int {
	if id, ok := l.nameIDs[name]; ok {
		return id
	}
	id := len(l.names)
	l.names = append(l.names, name)
	l.nameIDs[name] = id
	return id
}

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
		key := fileAttributeKey{
			FileEvidence: FileEvidence{Owner: c.Owner, Source: c.Source, Role: c.Role, Boundary: c.Boundary},
			class:        c.Class, metadata: c.Metadata,
		}
		id := l.lastAttribute
		if id == 0 || l.attributes[id-1].fileAttributeKey != key {
			var ok bool
			id, ok = l.attributeIDs[key]
			if !ok {
				id = len(l.attributes) + 1
				l.attributes = append(l.attributes, fileAttributes{fileAttributeKey: key, owner: l.nameID(c.Owner), source: l.nameID(c.Source)})
				l.attributeIDs[key] = id
			}
			l.lastAttribute = id
		}
		l.events = append(l.events, fileEvent{at: c.Offset, attribute: id}, fileEvent{at: c.Offset + c.Size, attribute: -id})
	}
	return nil
}

// fileCounts keeps a dense list of active IDs alongside direct counters.
// Iteration costs depend on the active set, not the number of packages or
// sources in the file, and endpoint updates do not hash strings.
type fileCounts struct {
	counts, positions, active []int
}

func newFileCounts(size int) fileCounts {
	return fileCounts{counts: make([]int, size), positions: make([]int, size)}
}

func (c *fileCounts) update(id, delta int) {
	if id == 0 {
		return
	}
	previous := c.counts[id]
	c.counts[id] += delta
	if previous == 0 {
		c.positions[id] = len(c.active)
		c.active = append(c.active, id)
	} else if c.counts[id] == 0 {
		index, last := c.positions[id], c.active[len(c.active)-1]
		c.active[index] = last
		c.positions[last] = index
		c.active = c.active[:len(c.active)-1]
	}
}

func (c *fileCounts) keys(names []string) []string {
	keys := make([]string, 0, len(c.active))
	for _, id := range c.active {
		keys = append(keys, names[id])
	}
	slices.Sort(keys)
	return keys
}

func (l *FileLedger) groupIDs() [][]int {
	groups := make([][]int, len(l.names))
	// Only claim owners need entries. Ancestors are interned as totals keys
	// even when they never appear as direct owners (including an empty name).
	for owner := range groups {
		if ancestors, ok := l.packageGroups[l.names[owner]]; ok {
			for _, ancestor := range ancestors {
				groups[owner] = append(groups[owner], l.nameID(ancestor))
			}
		} else {
			groups[owner] = []int{owner}
		}
	}
	return groups
}

// Finish partitions physical bytes by evidence priority. Ownership is counted
// as a set, so duplicate claims cannot inflate unique or shared coverage.
func (l *FileLedger) Finish(details bool) *FileCoverage {
	events := l.events
	slices.SortFunc(events, func(a, b fileEvent) int { return cmp.Compare(a.at, b.at) })
	groups := l.groupIDs()
	definitions, references, metadata := newFileCounts(len(l.names)), newFileCounts(len(l.names)), newFileCounts(len(l.names))
	evidence := map[FileEvidence]int{}
	sources := [3]fileCounts{newFileCounts(len(l.names)), newFileCounts(len(l.names)), newFileCounts(len(l.names))}
	counts := [3]int{}
	result := &FileCoverage{Version: 2, BySource: map[string]uint64{}, PackageSizes: map[string]uint64{}, MetadataSizes: map[string]uint64{}}
	packageSizes, metadataSizes, bySource := make([]uint64, len(l.names)), make([]uint64, len(l.names)), make([]uint64, len(l.names))
	groupSeen := make([]uint64, len(l.names))
	var previous uint64
	for i := 0; i < len(events); {
		at := events[i].at
		if at > previous {
			length := at - previous
			class := 0
			name := "unclassified"
			owners := &definitions
			if len(owners.active) == 0 {
				owners = &references
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
				for _, source := range sources[2].active {
					bySource[source] += length
				}
				if len(owners.active) > 1 {
					result.Shared += length
				}
				for _, owner := range owners.active {
					if metadata.counts[owner] > 0 {
						metadataSizes[owner] += length
					}
					for _, group := range groups[owner] {
						if groupSeen[group] != at {
							groupSeen[group] = at
							packageSizes[group] += length
						}
					}
				}
			} else if counts[1] > 0 {
				class = 1
				name = "recognized"
				result.Recognized += at - previous
			} else {
				result.Unclassified += at - previous
			}
			if details {
				region := FileRegion{FileRange: FileRange{Offset: previous, Size: at - previous}, Class: name, Sources: sources[class].keys(l.names)}
				if class == 2 {
					region.Owners = owners.keys(l.names)
					region.DefinitionOwners = definitions.keys(l.names)
					region.Referrers = references.keys(l.names)
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
			if e.attribute != 0 {
				id, delta := e.attribute, 1
				if id < 0 {
					id, delta = -id, -1
				}
				c := &l.attributes[id-1]
				counts[c.class] += delta
				if details || c.class == 2 {
					sources[c.class].update(c.source, delta)
				}
				if c.class == 2 {
					if c.Role == "reference" {
						references.update(c.owner, delta)
					} else {
						definitions.update(c.owner, delta)
					}
					if c.metadata {
						metadata.update(c.owner, delta)
					}
					if details {
						updateFileEvidence(evidence, c.FileEvidence, delta)
					}
				}
			}
			i++
		}
		previous = at
	}
	for id, name := range l.names {
		if packageSizes[id] > 0 {
			result.PackageSizes[name] = packageSizes[id]
		}
		if metadataSizes[id] > 0 {
			result.MetadataSizes[name] = metadataSizes[id]
		}
		if bySource[id] > 0 {
			result.BySource[name] = bySource[id]
		}
	}
	return result
}

func updateFileEvidence(evidence map[FileEvidence]int, item FileEvidence, delta int) {
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
