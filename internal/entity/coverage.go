package entity

import (
	"cmp"
	"fmt"
	"github.com/samber/lo"
	"slices"
)

// AddrCoverage is a list of AddrPos, describe the disAsmCoverage of the address space
type AddrCoverage []AddrPos

// CleanCoverage merge the overlapped AddrPos
func CleanCoverage(dirty AddrCoverage) AddrCoverage {
	dirty = lo.Uniq(dirty)

	slices.SortFunc(dirty, func(a, b AddrPos) int {
		if a.Addr != b.Addr {
			return cmp.Compare(a.Addr, b.Addr)
		}
		return cmp.Compare(a.Size, b.Size)
	})

	cover := make([]AddrPos, 0)
	for _, pos := range dirty {
		if len(cover) == 0 {
			cover = append(cover, pos)
			continue
		}

		last := &cover[len(cover)-1]
		if last.Addr+last.Size >= pos.Addr {
			// merge
			if last.Type != pos.Type {
				panic(fmt.Errorf("addr %x type %s and %s conflict", pos.Addr, last.Type, pos.Type))
			}

			if last.Addr+last.Size < pos.Addr+pos.Size {
				last.Size = pos.Addr + pos.Size - last.Addr
			}
		} else {
			cover = append(cover, pos)
		}
	}

	return cover
}

func MergeCoverage(coves ...AddrCoverage) AddrCoverage {
	size := 0
	for _, cov := range coves {
		size += len(cov)
	}

	ranges := make([]AddrPos, 0, size)
	for _, cov := range coves {
		ranges = append(ranges, cov...)
	}

	return CleanCoverage(ranges)
}
