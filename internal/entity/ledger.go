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
	Class   string   `json:"class"`
	Owners  []string `json:"owners,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

type FileCoverage struct {
	Attributed   uint64            `json:"attributed"`
	Recognized   uint64            `json:"recognized"`
	Unclassified uint64            `json:"unclassified"`
	Shared       uint64            `json:"shared"`
	BySource     map[string]uint64 `json:"by_source"`
	Regions      []FileRegion      `json:"regions,omitempty"`
}

type FileClaim struct {
	FileRange
	Owner  string
	Source string
	// 0: unclassified section, 1: recognized structure, 2: attributed bytes.
	Class int
}

type FileLedger struct {
	size   uint64
	claims []FileClaim
}

func NewFileLedger(size uint64) *FileLedger { return &FileLedger{size: size} }

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
	owners := map[string]int{}
	sources := [3]map[string]int{{}, {}, {}}
	counts := [3]int{}
	result := &FileCoverage{BySource: map[string]uint64{}}
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
			class := 0
			name := "unclassified"
			if counts[2] > 0 {
				class = 2
				name = "attributed"
				result.Attributed += at - previous
				for source := range sources[2] {
					result.BySource[source] += at - previous
				}
				if len(owners) > 1 {
					result.Shared += at - previous
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
				}
				if class == 0 && len(region.Sources) == 0 {
					region.Sources = []string{"unmapped-file-range"}
				}
				if len(result.Regions) > 0 {
					last := &result.Regions[len(result.Regions)-1]
					if last.Class == region.Class && slices.Equal(last.Owners, region.Owners) && slices.Equal(last.Sources, region.Sources) {
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
					update(owners, c.Owner, e.delta)
				}
			}
			i++
		}
		previous = at
	}
	return result
}
