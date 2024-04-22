package entity

import (
	"cmp"
	"fmt"
	"slices"
)

// AddrCoverage is a list of AddrPos, describe the disAsmCoverage of the address space
type AddrCoverage []*CoveragePart

type CoveragePart struct {
	Pos   AddrPos
	Addrs []*Addr
}

type ErrAddrCoverageConflict struct {
	Addr uint64
	Pos1 *CoveragePart
	Pos2 *CoveragePart
}

func (e ErrAddrCoverageConflict) Error() string {
	return fmt.Sprintf("addr %x pos %#v and %#v conflict", e.Addr, e.Pos1, e.Pos2)
}

// MergeCoverage merge multiple AddrCoverage
func MergeCoverage(coves []AddrCoverage) (AddrCoverage, *ErrAddrCoverageConflict) {
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
	for _, pos := range dirty {
		if len(cover) == 0 {
			cover = append(cover, pos)
			continue
		}

		last := cover[len(cover)-1]
		lastPos := last.Pos
		cur := pos.Pos

		if cur.Addr < lastPos.Addr+lastPos.Size {
			// merge
			if lastPos.Type != pos.Pos.Type {
				// if any is disasm, throw it
				if last.Addrs[len(last.Addrs)-1].SourceType == AddrSourceDisasm {
					cover = cover[:len(cover)-1]
				}
				if pos.Addrs[0].SourceType == AddrSourceDisasm {
					continue
				}

				return nil, &ErrAddrCoverageConflict{
					Addr: pos.Pos.Addr,
					Pos1: last,
					Pos2: pos,
				}
			}

			curEnd := pos.Pos.Addr + pos.Pos.Size
			if lastPos.Addr+lastPos.Size < curEnd {
				lastPos.Size = curEnd - lastPos.Addr
				last.Addrs = append(last.Addrs, pos.Addrs...)
			}
		} else {
			cover = append(cover, &CoveragePart{
				Pos:   pos.Pos,
				Addrs: pos.Addrs,
			})
		}
	}

	return cover, nil
}
