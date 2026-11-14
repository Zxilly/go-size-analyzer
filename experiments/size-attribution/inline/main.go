// Command inline is an experimental DWARF source-attribution overlay.
// It deliberately does not alter GSA's physical byte accounting.
package main

import (
	"cmp"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type span struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}
type (
	mapping      struct{ Address, Offset, Size uint64 }
	nativeSymbol struct {
		Name          string
		Address, Size uint64
		Function      bool
	}
)

type binaryData struct {
	DWARF      *dwarf.Data
	DWARFError error
	Format     string
	Maps       []mapping
	CodeMaps   []mapping
	Symbols    []nativeSymbol
	Close      func() error
}
type node struct {
	Entry               *dwarf.Entry
	Parent, Unit, Depth int
}
type occurrence struct {
	Name          string `json:"name"`
	Origin        string `json:"origin"`
	Host          string `json:"host"`
	CallSite      string `json:"call_site,omitempty"`
	Depth         int    `json:"depth"`
	AddressRanges []span `json:"address_ranges"`
	FileRanges    []span `json:"file_ranges"`
}
type sourceGroup struct {
	Origin         string   `json:"origin"`
	Occurrences    int      `json:"occurrences"`
	InclusiveBytes uint64   `json:"inclusive_union_bytes"`
	DeepestBytes   uint64   `json:"deepest_source_bytes"`
	Hosts          []string `json:"hosts"`
}
type genericBody struct {
	Name        string `json:"name"`
	Declaration string `json:"declaration"`
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	FileRanges  []span `json:"file_ranges"`
	Bytes       uint64 `json:"union_bytes"`
}
type genericGroup struct {
	Declaration string        `json:"declaration"`
	Bodies      []genericBody `json:"emitted_bodies"`
	Bytes       uint64        `json:"union_bytes"`
}
type report struct {
	File                     string         `json:"file"`
	SHA256                   string         `json:"sha256"`
	FileSize                 int64          `json:"file_size"`
	Format                   string         `json:"format"`
	GoVersion                string         `json:"go_version,omitempty"`
	DWARFAvailable           bool           `json:"dwarf_available"`
	ConcreteFunctions        int            `json:"concrete_functions"`
	InlineOccurrences        int            `json:"inline_occurrences"`
	UnresolvedInlineNames    int            `json:"unresolved_inline_names"`
	UnmappedRangeBytes       uint64         `json:"unmapped_range_bytes"`
	InlineUnionBytes         uint64         `json:"inline_union_bytes"`
	DeepestAmbiguousBytes    uint64         `json:"deepest_ambiguous_bytes"`
	Sources                  []sourceGroup  `json:"sources"`
	Inlines                  []occurrence   `json:"inlines"`
	GenericGroups            []genericGroup `json:"generic_groups"`
	GenericSource            string         `json:"generic_source"`
	GenericUnrecognizedNames []string       `json:"generic_unrecognized_names"`
	DictionarySymbols        []string       `json:"dictionary_symbols"`
	Notes                    []string       `json:"notes"`
}

func main() {
	output := flag.String("output", "", "JSON output path; stdout when omitted")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: inline [-output report.json] BINARY...")
		os.Exit(2)
	}
	reports := make([]report, 0, flag.NArg())
	for _, path := range flag.Args() {
		r, err := analyze(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		reports = append(reports, r)
	}
	var out io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		out = f
	}
	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	if err := e.Encode(reports); err != nil {
		panic(err)
	}
}

func openBinary(path string) (*binaryData, error) {
	if f, err := elf.Open(path); err == nil {
		b := &binaryData{Format: "ELF", Close: f.Close}
		for _, s := range f.Sections {
			if s.Type != elf.SHT_NOBITS && s.Addr != 0 {
				b.Maps = append(b.Maps, mapping{s.Addr, s.Offset, s.FileSize})
				if s.Flags&elf.SHF_EXECINSTR != 0 {
					b.CodeMaps = append(b.CodeMaps, mapping{s.Addr, s.Offset, s.FileSize})
				}
			}
		}
		if syms, err := f.Symbols(); err == nil {
			for _, s := range syms {
				b.Symbols = append(b.Symbols, nativeSymbol{s.Name, s.Value, s.Size, elf.ST_TYPE(s.Info) == elf.STT_FUNC && s.Section != elf.SHN_UNDEF})
			}
		}
		b.DWARF, b.DWARFError = f.DWARF()
		return b, nil
	}
	if f, err := pe.Open(path); err == nil {
		b := &binaryData{Format: "PE", Close: f.Close}
		var base uint64
		switch h := f.OptionalHeader.(type) {
		case *pe.OptionalHeader64:
			base = h.ImageBase
		case *pe.OptionalHeader32:
			base = uint64(h.ImageBase)
		default:
			_ = f.Close()
			return nil, errors.New("unsupported PE optional header")
		}
		for _, s := range f.Sections {
			b.Maps = append(b.Maps, mapping{base + uint64(s.VirtualAddress), uint64(s.Offset), uint64(s.Size)})
		}
		for _, s := range f.Symbols {
			if s.SectionNumber > 0 && int(s.SectionNumber) <= len(f.Sections) {
				sec := f.Sections[s.SectionNumber-1]
				b.Symbols = append(b.Symbols, nativeSymbol{Name: s.Name, Address: base + uint64(sec.VirtualAddress) + uint64(s.Value)})
			}
		}
		b.DWARF, b.DWARFError = f.DWARF()
		return b, nil
	}
	if f, err := macho.Open(path); err == nil {
		b := &binaryData{Format: "Mach-O", Close: f.Close}
		for _, s := range f.Sections {
			if s.Offset != 0 {
				b.Maps = append(b.Maps, mapping{s.Addr, uint64(s.Offset), s.Size})
			}
		}
		if f.Symtab != nil {
			for _, s := range f.Symtab.Syms {
				b.Symbols = append(b.Symbols, nativeSymbol{Name: strings.TrimPrefix(s.Name, "_"), Address: s.Value})
			}
		}
		b.DWARF, b.DWARFError = f.DWARF()
		return b, nil
	}
	return nil, errors.New("unsupported or unreadable binary (ELF/PE/thin Mach-O only)")
}

func analyze(path string) (report, error) {
	r := report{File: path, Notes: []string{
		"Inline ranges are a source overlay on existing machine-code bytes, not additional bytes.",
		"Inclusive source groups overlap for nested inlining and cannot be summed; deepest source assigns each unambiguous byte once.",
		"Generic groups describe emitted DWARF or sized ELF function-symbol bodies, not source-level instantiation counts; go.shape bodies may be shared.",
		"Dictionary symbols are listed separately and excluded from code-size totals; no dictionary-size inference is made.",
	}}
	f, err := os.Open(path)
	if err != nil {
		return r, err
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	closeErr := f.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return r, err
	}
	r.SHA256 = hex.EncodeToString(h.Sum(nil))
	st, err := os.Stat(path)
	if err != nil {
		return r, err
	}
	r.FileSize = st.Size()
	if bi, err := buildinfo.ReadFile(path); err == nil {
		r.GoVersion = bi.GoVersion
	}
	b, err := openBinary(path)
	if err != nil {
		return r, err
	}
	defer func() { _ = b.Close() }() // Read-only input; no buffered writes to flush.
	r.Format = b.Format
	for _, s := range b.Symbols {
		if strings.Contains(s.Name, "..dict.") {
			r.DictionarySymbols = append(r.DictionarySymbols, s.Name)
		}
	}
	slices.Sort(r.DictionarySymbols)
	if b.DWARF == nil {
		r.Notes = append(r.Notes, "DWARF unavailable or unsupported by the standard-library reader; no inline reconstruction attempted.")
		if b.DWARFError != nil {
			r.Notes = append(r.Notes, "DWARF reader: "+b.DWARFError.Error())
		}
		nativeGenericFallback(&r, b)
		return r, nil
	}
	nodes, index, files, err := readNodes(b.DWARF)
	if err != nil {
		r.Notes = append(r.Notes, "DWARF decode failed: "+err.Error())
		nativeGenericFallback(&r, b)
		return r, nil
	}
	if len(nodes) == 0 {
		r.Notes = append(r.Notes, "DWARF has no entries; no inline reconstruction attempted.")
		nativeGenericFallback(&r, b)
		return r, nil
	}
	r.DWARFAvailable = true
	r.GenericSource = "dwarf"
	gbody := map[string]*genericBody{}
	for i, n := range nodes {
		if n.Entry.Tag != dwarf.TagSubprogram && n.Entry.Tag != dwarf.TagInlinedSubroutine {
			continue
		}
		raw, err := b.DWARF.Ranges(n.Entry)
		if err != nil {
			r.Notes = append(r.Notes, fmt.Sprintf("range decode failed at DIE %#x: %v", n.Entry.Offset, err))
			continue
		}
		var addr []span
		for _, pair := range raw {
			if pair[1] > pair[0] {
				addr = append(addr, span{pair[0], pair[1]})
			}
		}
		addr = union(addr)
		if len(addr) == 0 {
			continue
		}
		mapped := mapRanges(addr, b.Maps, uint64(r.FileSize))
		r.UnmappedRangeBytes += size(addr) - size(mapped)
		name := inheritedString(i, dwarf.AttrName, nodes, index)
		if n.Entry.Tag == dwarf.TagSubprogram {
			r.ConcreteFunctions++
			if decl, kind, ok := genericDeclaration(name); ok {
				body := gbody[name]
				if body == nil {
					body = &genericBody{Name: name, Declaration: decl, Kind: kind, Source: "dwarf"}
					gbody[name] = body
				}
				body.FileRanges = append(body.FileRanges, mapped...)
			} else if strings.Contains(name, "[") || strings.Contains(name, "]") {
				r.GenericUnrecognizedNames = append(r.GenericUnrecognizedNames, name)
			}
			continue
		}
		r.InlineOccurrences++
		if name == "" {
			r.UnresolvedInlineNames++
			name = fmt.Sprintf("<unresolved DIE %#x>", n.Entry.Offset)
		}
		host, depth := "", 1
		for p := n.Parent; p >= 0; p = nodes[p].Parent {
			if nodes[p].Entry.Tag == dwarf.TagInlinedSubroutine {
				depth++
			}
			if nodes[p].Entry.Tag == dwarf.TagSubprogram {
				host = inheritedString(p, dwarf.AttrName, nodes, index)
				break
			}
		}
		origin := originKey(i, name, nodes, index, files)
		call := location(n.Unit, n.Entry.Val(dwarf.AttrCallFile), n.Entry.Val(dwarf.AttrCallLine), files)
		r.Inlines = append(r.Inlines, occurrence{Name: name, Origin: origin, Host: host, CallSite: call, Depth: depth, AddressRanges: addr, FileRanges: mapped})
	}
	r.Sources, r.InlineUnionBytes, r.DeepestAmbiguousBytes = summarizeSources(r.Inlines)
	finishGenericGroups(&r, gbody)
	return r, nil
}

func nativeGenericFallback(r *report, b *binaryData) {
	r.GenericSource = "unavailable"
	if b.Format != "ELF" {
		r.Notes = append(r.Notes, "Native generic fallback requires sized ELF STT_FUNC definitions; PE/Mach-O symbol-size inference is unsupported.")
		return
	}
	r.GenericSource = "native_symbol"
	r.Notes = append(r.Notes, "Generic fallback uses sized ELF STT_FUNC definitions fully mapped into executable file sections; it provides no inline information.")
	bodies := map[string]*genericBody{}
	for _, s := range b.Symbols {
		if !s.Function || s.Size == 0 || s.Address+s.Size < s.Address {
			continue
		}
		decl, kind, ok := genericDeclaration(s.Name)
		if !ok {
			if strings.ContainsAny(s.Name, "[]") {
				r.GenericUnrecognizedNames = append(r.GenericUnrecognizedNames, s.Name)
			}
			continue
		}
		mapped := mapRanges([]span{{s.Address, s.Address + s.Size}}, b.CodeMaps, uint64(r.FileSize))
		if size(mapped) != s.Size {
			continue
		}
		body := bodies[s.Name]
		if body == nil {
			body = &genericBody{Name: s.Name, Declaration: decl, Kind: kind, Source: "native_symbol"}
			bodies[s.Name] = body
		}
		body.FileRanges = append(body.FileRanges, mapped...)
	}
	finishGenericGroups(r, bodies)
}

func finishGenericGroups(r *report, gbody map[string]*genericBody) {
	groups := map[string]*genericGroup{}
	for _, body := range gbody {
		body.FileRanges = union(body.FileRanges)
		body.Bytes = size(body.FileRanges)
		g := groups[body.Declaration]
		if g == nil {
			g = &genericGroup{Declaration: body.Declaration}
			groups[body.Declaration] = g
		}
		g.Bodies = append(g.Bodies, *body)
	}
	for _, g := range groups {
		var ranges []span
		for _, body := range g.Bodies {
			ranges = append(ranges, body.FileRanges...)
		}
		g.Bytes = size(union(ranges))
		slices.SortFunc(g.Bodies, func(a, b genericBody) int { return cmp.Compare(a.Name, b.Name) })
		r.GenericGroups = append(r.GenericGroups, *g)
	}
	slices.SortFunc(r.GenericGroups, func(a, b genericGroup) int {
		if n := cmp.Compare(b.Bytes, a.Bytes); n != 0 {
			return n
		}
		return cmp.Compare(a.Declaration, b.Declaration)
	})
	slices.Sort(r.GenericUnrecognizedNames)
}

func readNodes(d *dwarf.Data) ([]node, map[dwarf.Offset]int, map[int][]*dwarf.LineFile, error) {
	var nodes []node
	index := map[dwarf.Offset]int{}
	files := map[int][]*dwarf.LineFile{}
	var stack []int
	unit := -1
	reader := d.Reader()
	for {
		e, err := reader.Next()
		if err != nil {
			return nil, nil, nil, err
		}
		if e == nil {
			break
		}
		if e.Tag == 0 {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		parent := -1
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		i := len(nodes)
		if e.Tag == dwarf.TagCompileUnit {
			unit = i
			if lr, err := d.LineReader(e); err == nil && lr != nil {
				files[unit] = lr.Files()
			}
		}
		nodes = append(nodes, node{Entry: e, Parent: parent, Unit: unit, Depth: len(stack)})
		index[e.Offset] = i
		if e.Children {
			stack = append(stack, i)
		}
	}
	return nodes, index, files, nil
}

func inherited(i int, attr dwarf.Attr, nodes []node, index map[dwarf.Offset]int) (any, int) {
	seen := map[int]bool{}
	for i >= 0 && !seen[i] {
		seen[i] = true
		if v := nodes[i].Entry.Val(attr); v != nil {
			return v, i
		}
		next := -1
		for _, link := range []dwarf.Attr{dwarf.AttrAbstractOrigin, dwarf.AttrSpecification} {
			if off, ok := nodes[i].Entry.Val(link).(dwarf.Offset); ok {
				if p, ok := index[off]; ok {
					next = p
					break
				}
			}
		}
		i = next
	}
	return nil, -1
}

func inheritedString(i int, attr dwarf.Attr, nodes []node, index map[dwarf.Offset]int) string {
	v, _ := inherited(i, attr, nodes, index)
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func intValue(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case uint64:
		if x <= 1<<63-1 {
			return int64(x)
		}
	case int:
		return int64(x)
	default:
		return -1
	}
	return -1
}

func location(unit int, file, line any, files map[int][]*dwarf.LineFile) string {
	n := intValue(file)
	fs := files[unit]
	if n < 0 || n >= int64(len(fs)) || fs[n] == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(fs[n].Name), intValue(line))
}

func originKey(i int, name string, nodes []node, index map[dwarf.Offset]int, files map[int][]*dwarf.LineFile) string {
	f, at := inherited(i, dwarf.AttrDeclFile, nodes, index)
	if at < 0 {
		return name
	}
	l, _ := inherited(i, dwarf.AttrDeclLine, nodes, index)
	loc := location(nodes[at].Unit, f, l, files)
	if loc == "" {
		return name
	}
	return name + " @ " + loc
}

func genericDeclaration(name string) (declaration, bodyKind string, valid bool) {
	if strings.Contains(name, "..dict.") || strings.HasPrefix(name, "type:") || strings.HasPrefix(name, "type.") {
		return "", "", false
	}
	depth, found := 0, false
	var b strings.Builder
	for _, c := range name {
		switch c {
		case '[':
			depth++
			found = true
		case ']':
			depth--
			if depth < 0 {
				return "", "", false
			}
		default:
			if depth == 0 {
				b.WriteRune(c)
			}
		}
	}
	if !found || depth != 0 {
		return "", "", false
	}
	kind := "symbol-specific-body"
	if strings.Contains(name, "go.shape.") {
		kind = "shared-shape-body"
	}
	return b.String(), kind, true
}

func union(in []span) []span {
	a := append([]span(nil), in...)
	slices.SortFunc(a, func(a, b span) int {
		if n := cmp.Compare(a.Start, b.Start); n != 0 {
			return n
		}
		return cmp.Compare(a.End, b.End)
	})
	var out []span
	for _, r := range a {
		if r.End <= r.Start {
			continue
		}
		if len(out) == 0 || r.Start > out[len(out)-1].End {
			out = append(out, r)
		} else if r.End > out[len(out)-1].End {
			out[len(out)-1].End = r.End
		}
	}
	return out
}

func size(ranges []span) uint64 {
	var n uint64
	for _, r := range ranges {
		n += r.End - r.Start
	}
	return n
}

func mapRanges(addr []span, maps []mapping, fileSize uint64) []span {
	var out []span
	for _, a := range addr {
		for _, m := range maps {
			if m.Address+m.Size < m.Address {
				continue
			}
			lo, hi := max(a.Start, m.Address), min(a.End, m.Address+m.Size)
			if hi <= lo {
				continue
			}
			off := m.Offset + (lo - m.Address)
			if off < m.Offset || off >= fileSize {
				continue
			}
			end := min(fileSize, off+(hi-lo))
			if end > off {
				out = append(out, span{off, end})
			}
		}
	}
	return union(out)
}

func summarizeSources(items []occurrence) (sourceGroups []sourceGroup, unionBytes, ambiguousBytes uint64) {
	type event struct {
		At   uint64
		Item int
		Add  bool
	}
	var events []event
	groups := map[string]*sourceGroup{}
	ranges := map[string][]span{}
	hosts := map[string]map[string]bool{}
	for i, item := range items {
		g := groups[item.Origin]
		if g == nil {
			g = &sourceGroup{Origin: item.Origin}
			groups[item.Origin] = g
			hosts[item.Origin] = map[string]bool{}
		}
		g.Occurrences++
		hosts[item.Origin][item.Host] = true
		ranges[item.Origin] = append(ranges[item.Origin], item.FileRanges...)
		for _, r := range item.FileRanges {
			events = append(events, event{r.Start, i, true}, event{r.End, i, false})
		}
	}
	slices.SortFunc(events, func(a, b event) int { return cmp.Compare(a.At, b.At) })
	active := map[int]int{}
	var total, ambiguous uint64
	for i := 0; i < len(events); {
		at := events[i].At
		for i < len(events) && events[i].At == at {
			e := events[i]
			if e.Add {
				active[e.Item]++
			} else {
				active[e.Item]--
				if active[e.Item] == 0 {
					delete(active, e.Item)
				}
			}
			i++
		}
		if i == len(events) || len(active) == 0 {
			continue
		}
		n := events[i].At - at
		total += n
		depth := -1
		source := ""
		conflict := false
		for id := range active {
			item := items[id]
			if item.Depth > depth {
				depth = item.Depth
				source = item.Origin
				conflict = false
			} else if item.Depth == depth && item.Origin != source {
				conflict = true
			}
		}
		if conflict {
			ambiguous += n
		} else {
			groups[source].DeepestBytes += n
		}
	}
	var out []sourceGroup
	for name, g := range groups {
		g.InclusiveBytes = size(union(ranges[name]))
		for h := range hosts[name] {
			g.Hosts = append(g.Hosts, h)
		}
		slices.Sort(g.Hosts)
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b sourceGroup) int {
		if n := cmp.Compare(b.DeepestBytes, a.DeepestBytes); n != 0 {
			return n
		}
		return cmp.Compare(a.Origin, b.Origin)
	})
	return out, total, ambiguous
}
