package entity

import (
	"cmp"
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

// MergeAndCleanCoverage merge multiple AddrCoverage
func MergeAndCleanCoverage(coves []AddrCoverage) (AddrCoverage, error) {
	size := 0
	for _, cov := range coves {
		size += len(cov)
	}

	dirty := make(AddrCoverage, 0, size)
	for _, cov := range coves {
		dirty = append(dirty, cov...)
	}

	slices.SortFunc(dirty, func(a, b *CoveragePart) int {
		if a.Pos.Addr != b.Pos.Addr {
			return cmp.Compare(a.Pos.Addr, b.Pos.Addr)
		}
		return cmp.Compare(a.Pos.Size, a.Pos.Size)
	})

	cover := make(AddrCoverage, 0)
	for _, curCov := range dirty {
		if len(cover) == 0 {
			cover = append(cover, curCov)
			continue
		}

		last := cover[len(cover)-1]
		lastPos := last.Pos
		curPos := curCov.Pos

		if curPos.Addr < lastPos.Addr+lastPos.Size {
			// merge
			if lastPos.Type != curCov.Pos.Type {
				processed := false
				// if any is disasm, throw it
				if last.HasDisasm() {
					cover = cover[:len(cover)-1]
					processed = true
				}
				if curCov.HasDisasm() {
					continue
				}
				if processed {
					cover = append(cover, curCov)
					continue
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
		} else {
			cover = append(cover, &CoveragePart{
				Pos:   curCov.Pos,
				Addrs: curCov.Addrs,
			})
		}
	}

	return cover, nil
}
