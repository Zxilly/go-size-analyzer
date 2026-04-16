package entity

import (
	"cmp"
	"container/heap"
	"fmt"
	"slices"
	"strings"
)

// AddrCoverage is a list of AddrPos, describe the coverage of the address space
type AddrCoverage []*CoveragePart

type CoveragePart struct {
	Pos   *AddrPos
	Addrs []*Addr
}

func (c *CoveragePart) HasDisasm() bool {
	for _, addr := range c.Addrs {
		if addr.SourceType == AddrSourceDisasm {
			return true
		}
	}
	return false
}

func (c *CoveragePart) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Pos: %s", c.Pos))
	for _, addr := range c.Addrs {
		parts = append(parts, addr.String())
	}
	return strings.Join(parts, "\n")
}

type ErrAddrCoverageConflict struct {
	Addr uint64
	Pos1 *CoveragePart
	Pos2 *CoveragePart
}

func (e *ErrAddrCoverageConflict) Error() string {
	return fmt.Sprintf("addr %x pos %s and %s conflict", e.Addr, e.Pos1, e.Pos2)
}

func compareCoveragePart(a, b *CoveragePart) int {
	if a.Pos.Addr != b.Pos.Addr {
		return cmp.Compare(a.Pos.Addr, b.Pos.Addr)
	}
	return cmp.Compare(a.Pos.Size, b.Pos.Size)
}

// mergeHeapItem represents one sorted coverage stream in the k-way merge
type mergeHeapItem struct {
	cov AddrCoverage
	idx int // current position in cov
}

type mergeHeap []mergeHeapItem

func (h mergeHeap) Len() int { return len(h) }
func (h mergeHeap) Less(i, j int) bool {
	return compareCoveragePart(h[i].cov[h[i].idx], h[j].cov[h[j].idx]) < 0
}
func (h mergeHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *mergeHeap) Push(x any) {
	*h = append(*h, x.(mergeHeapItem))
}

func (h *mergeHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// kWayMerge merges k sorted AddrCoverages into one sorted stream.
// Each input coverage must be pre-sorted by (Addr, Size).
func kWayMerge(coves []AddrCoverage) AddrCoverage {
	totalSize := 0
	h := &mergeHeap{}

	for _, cov := range coves {
		if len(cov) == 0 {
			continue
		}
		totalSize += len(cov)
		heap.Push(h, mergeHeapItem{cov: cov, idx: 0})
	}

	result := make(AddrCoverage, 0, totalSize)

	for h.Len() > 0 {
		item := heap.Pop(h).(mergeHeapItem)
		result = append(result, item.cov[item.idx])
		item.idx++
		if item.idx < len(item.cov) {
			heap.Push(h, item)
		}
	}

	return result
}

// MergeAndCleanCoverage merge multiple AddrCoverage
func MergeAndCleanCoverage(coves []AddrCoverage) (AddrCoverage, error) {
	// Sort each individual coverage that isn't already sorted,
	// then use k-way merge for efficient merging
	for i, cov := range coves {
		if !slices.IsSortedFunc(cov, compareCoveragePart) {
			sorted := make(AddrCoverage, len(cov))
			copy(sorted, cov)
			slices.SortFunc(sorted, compareCoveragePart)
			coves[i] = sorted
		}
	}

	dirty := kWayMerge(coves)

	cover := make(AddrCoverage, 0)
	for _, curCov := range dirty {
		// Re-loop after dropping a tainted last so curCov is re-compared with
		// the newly exposed predecessor.
		for {
			if len(cover) == 0 {
				cover = append(cover, curCov)
				break
			}

			last := cover[len(cover)-1]
			lastPos := last.Pos
			curPos := curCov.Pos

			if curPos.Addr >= lastPos.Addr+lastPos.Size {
				cover = append(cover, &CoveragePart{
					Pos:   curCov.Pos,
					Addrs: curCov.Addrs,
				})
				break
			}

			if lastPos.Type != curCov.Pos.Type {
				if last.HasDisasm() {
					cover = cover[:len(cover)-1]
					if curCov.HasDisasm() {
						break
					}
					continue
				}
				if curCov.HasDisasm() {
					break
				}
				return nil, &ErrAddrCoverageConflict{
					Addr: curCov.Pos.Addr,
					Pos1: last,
					Pos2: curCov,
				}
			}

			curEnd := curCov.Pos.Addr + curCov.Pos.Size
			if lastPos.Addr+lastPos.Size < curEnd {
				lastPos.Size = curEnd - lastPos.Addr
				last.Addrs = append(last.Addrs, curCov.Addrs...)
			}
			break
		}
	}

	return cover, nil
}
