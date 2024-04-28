package entity

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type AddrPos struct {
	Addr uint64
	Size uint64
	Type AddrType
}

func (a *AddrPos) String() string {
	return fmt.Sprintf("Addr: %x Size: %x Type: %s", a.Addr, a.Size, a.Type)
}

type Addr struct {
	*AddrPos

	Pkg      *Package  // package can be nil for cgo symbols
	Function *Function // for symbol source it will be a nil

	SourceType AddrSourceType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: 0x%x Size: %d pkg: %s SourceType: %s", a.Addr, a.Size, a.Pkg.Name, a.SourceType)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

// AddrCoverage is a list of AddrPos, describe the coverage of the address space
type AddrCoverage []*CoveragePart

type CoveragePart struct {
	Pos   *AddrPos
	Addrs []*Addr
}

func (c *CoveragePart) String() string {
	sb := new(strings.Builder)
	sb.WriteString(fmt.Sprintf("Pos: %s", c.Pos))
	for _, addr := range c.Addrs {
		sb.WriteString(fmt.Sprintf("  %s", addr))
	}
	return sb.String()
}

type ErrAddrCoverageConflict struct {
	Addr uint64
	Pos1 *CoveragePart
	Pos2 *CoveragePart
}

func (e *ErrAddrCoverageConflict) Error() string {
	return fmt.Sprintf("addr %x pos %#v and %#v conflict", e.Addr, e.Pos1, e.Pos2)
}

func CleanCoverage(cov AddrCoverage) (AddrCoverage, error) {
	return MergeAndCleanCoverage([]AddrCoverage{cov})
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
