package dwarf

import "debug/dwarf"

// CompileUnitEntries contains the entries consumed by package attribution.
// Function children are excluded, but still visited when reading Go type kinds.
type CompileUnitEntries struct {
	Unit    *dwarf.Entry
	Entries []*dwarf.Entry
}

// AnalysisEntries shares the Go kind scan with the later attribution pass.
// CanReplay is false when the original reader must handle unusual structure or
// sibling offsets. Kinds remains complete even when replay is unavailable.
type AnalysisEntries struct {
	Kinds     map[dwarf.Type]int64
	Units     []CompileUnitEntries
	CanReplay bool
}

// ReadAnalysisEntries decodes every DIE before attribution, preserving failures
// in entries that the attribution reader would otherwise skip. Only the small
// subset used by attribution is retained for replay after type kinds are known.
func ReadAnalysisEntries(d *dwarf.Data) (*AnalysisEntries, error) {
	collector := entryCollector{skipDepth: -1}
	kinds, err := scanGoTypeKinds(d, collector.add)
	if err != nil {
		return nil, err
	}
	if collector.depth != 0 {
		collector.fallback()
	}
	return &AnalysisEntries{Kinds: kinds, Units: collector.units, CanReplay: !collector.legacy}, nil
}

type entryCollector struct {
	units     []CompileUnitEntries
	depth     int
	skipDepth int
	legacy    bool
}

func (c *entryCollector) fallback() {
	c.legacy = true
	c.units = nil
}

func (c *entryCollector) add(entry *dwarf.Entry) {
	if c.legacy {
		return
	}
	// Reader.SkipChildren follows sibling offsets even within a skipped
	// subtree. Retain the original reader for those files rather than infer
	// the target or suppress errors caused by an invalid sibling offset.
	if entry.Children && entry.Val(dwarf.AttrSibling) != nil {
		c.fallback()
		return
	}
	if entry.Tag == 0 {
		if c.depth == 0 {
			c.fallback()
			return
		}
		c.depth--
		if c.depth == c.skipDepth {
			c.skipDepth = -1
		}
		return
	}
	if c.depth == 0 && entry.Tag != dwarf.TagCompileUnit {
		c.fallback()
		return
	}
	if entry.Tag == dwarf.TagCompileUnit {
		if c.depth != 0 {
			c.fallback()
			return
		}
		c.units = append(c.units, CompileUnitEntries{Unit: entry})
	} else if c.skipDepth == -1 && (entry.Tag == dwarf.TagSubprogram || entry.Tag == dwarf.TagVariable) {
		unit := &c.units[len(c.units)-1]
		unit.Entries = append(unit.Entries, entry)
		if entry.Tag == dwarf.TagSubprogram && entry.Children {
			c.skipDepth = c.depth
		}
	}
	if entry.Children {
		c.depth++
	}
}
