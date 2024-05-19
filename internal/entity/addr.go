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
	return fmt.Sprintf("Addr: %x CodeSize: %x Type: %s", a.Addr, a.Size, a.Type)
}

type Addr struct {
	*AddrPos

	Pkg      *Package  // package can be nil for cgo symbols
	Function *Function // for symbol source it will be a nil

	SourceType AddrSourceType

	Meta any
}

func (a *Addr) String() string {
	var pkgName, funcName string
	if a.Pkg != nil {
		pkgName = a.Pkg.Name
	}
	if a.Function != nil {
		funcName = a.Function.Name
	}
	return fmt.Sprintf("AddrPos: %s Pkg: %s Function: %s SourceType: %s", a.AddrPos, pkgName, funcName, a.SourceType)
}

// AddrCoverage is a list of AddrPos, describe the coverage of the address space
type AddrCoverage []*CoveragePart

type CoveragePart struct {
	Pos   *AddrPos
	Addrs []*Addr
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
				processed := false
				// if any is disasm, throw it
				if last.Addrs[len(last.Addrs)-1].SourceType == AddrSourceDisasm {
					cover = cover[:len(cover)-1]
					processed = true
				}
				if pos.Addrs[0].SourceType == AddrSourceDisasm {
					continue
				}
				if processed {
					cover = append(cover, pos)
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
