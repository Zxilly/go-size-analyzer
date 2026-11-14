// Command staticreach is a deliberately bounded experiment, not a production analyzer.
package main

import (
	"cmp"
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/ZxillyFork/gore"
)

type span struct {
	Offset uint64 `json:"offset"`
	Size   uint64 `json:"size"`
}
type candidate struct {
	span
	Address    uint64 `json:"address"`
	Root       string `json:"root"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	CrossBlock bool   `json:"cross_block,omitempty"`
}
type summary struct {
	Candidates        int         `json:"candidate_count"`
	Unique            uint64      `json:"unique_file_bytes"`
	AlreadyAttributed uint64      `json:"already_attributed_bytes"`
	New               uint64      `json:"new_reference_bytes"`
	NewRecognized     uint64      `json:"new_from_recognized_bytes"`
	NewUnclassified   uint64      `json:"new_from_unclassified_bytes"`
	NewRanges         []span      `json:"new_ranges"`
	Examples          []candidate `json:"examples"`
	NewExamples       []candidate `json:"new_examples"`
}
type report struct {
	Input            string                                                `json:"input"`
	FileSize         uint64                                                `json:"file_size"`
	Baseline         struct{ Attributed, Recognized, Unclassified uint64 } `json:"baseline"`
	Static           summary                                               `json:"typed_static_references"`
	Interfaces       summary                                               `json:"interface_candidates"`
	CFG              summary                                               `json:"cfg_string_candidates"`
	CFGCrossBlock    summary                                               `json:"cfg_cross_block_candidates"`
	Roots            int                                                   `json:"dwarf_roots"`
	Cycles           int                                                   `json:"cycle_or_shared_revisits"`
	Functions        int                                                   `json:"functions_scanned"`
	SkippedFunctions int                                                   `json:"functions_over_budget"`
	Notes            []string                                              `json:"notes"`
}
type imageFile struct {
	file *elf.File
	data []byte
}

func (m imageFile) mapAddr(addr, size uint64) (span, bool) {
	if size == 0 || addr+size < addr {
		return span{}, false
	}
	for _, p := range m.file.Progs {
		if p.Type == elf.PT_LOAD && addr >= p.Vaddr && addr-p.Vaddr <= p.Filesz && size <= p.Filesz-(addr-p.Vaddr) {
			return span{p.Off + addr - p.Vaddr, size}, true
		}
	}
	return span{}, false
}

func (m imageFile) read(addr, size uint64) ([]byte, bool) {
	s, ok := m.mapAddr(addr, size)
	if !ok || s.Offset+size > uint64(len(m.data)) {
		return nil, false
	}
	return m.data[s.Offset : s.Offset+size], true
}

func (m imageFile) ptr(addr uint64) (uint64, bool) {
	b, ok := m.read(addr, 8)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint64(b), true
}

func union(in []span) []span {
	in = append([]span(nil), in...)
	slices.SortFunc(in, func(a, b span) int { return cmp.Compare(a.Offset, b.Offset) })
	out := []span{}
	for _, s := range in {
		if s.Size == 0 {
			continue
		}
		if len(out) > 0 && s.Offset <= out[len(out)-1].Offset+out[len(out)-1].Size {
			p := &out[len(out)-1]
			p.Size = max(p.Offset+p.Size, s.Offset+s.Size) - p.Offset
		} else {
			out = append(out, s)
		}
	}
	return out
}

func subtract(a, b []span) []span {
	a = union(a)
	b = union(b)
	out := []span{}
	j := 0
	for _, s := range a {
		p, end := s.Offset, s.Offset+s.Size
		for j < len(b) && b[j].Offset+b[j].Size <= p {
			j++
		}
		for k := j; k < len(b) && b[k].Offset < end; k++ {
			if b[k].Offset > p {
				out = append(out, span{p, b[k].Offset - p})
			}
			p = max(p, b[k].Offset+b[k].Size)
			if p >= end {
				break
			}
		}
		if p < end {
			out = append(out, span{p, end - p})
		}
	}
	return out
}

func total(a []span) uint64 {
	var n uint64
	for _, s := range a {
		n += s.Size
	}
	return n
}

func summarize(cs []candidate, baseline []span, recognized ...[]span) summary {
	spans := []span{}
	for _, c := range cs {
		spans = append(spans, c.span)
	}
	u := union(spans)
	fresh := subtract(u, baseline)
	s := summary{Candidates: len(cs), Unique: total(u), New: total(fresh), NewRanges: fresh}
	s.AlreadyAttributed = s.Unique - s.New
	s.NewUnclassified = s.New
	if len(recognized) > 0 {
		s.NewUnclassified = total(subtract(fresh, recognized[0]))
		s.NewRecognized = s.New - s.NewUnclassified
	}
	slices.SortFunc(cs, func(a, b candidate) int { return cmp.Compare(b.Size, a.Size) })
	s.Examples = cs[:min(30, len(cs))]
	for _, c := range cs {
		at := sort.Search(len(fresh), func(i int) bool { return fresh[i].Offset+fresh[i].Size > c.Offset })
		if at < len(fresh) && fresh[at].Offset < c.Offset+c.Size {
			s.NewExamples = append(s.NewExamples, c)
			if len(s.NewExamples) == 30 {
				break
			}
		}
	}
	return s
}

func underlying(t dwarf.Type) dwarf.Type {
	for i := 0; i < 32; i++ {
		switch x := t.(type) {
		case *dwarf.TypedefType:
			t = x.Type
		case *dwarf.QualType:
			t = x.Type
		default:
			return t
		}
	}
	return nil
}

type visitKey struct {
	addr  uint64
	typ   dwarf.Type
	iface bool
}
type walker struct {
	m                  imageFile
	seen               map[visitKey]bool
	types              map[string]dwarf.Type
	kinds              map[dwarf.Type]int64
	runtimeTypes       map[uint64]*gore.GoType
	static, interfaces []candidate
	cycles, steps      int
	follows            map[dwarf.Type]bool
}

func (w *walker) canFollow(t dwarf.Type) bool {
	if t == nil {
		return false
	}
	if w.follows == nil {
		w.follows = map[dwarf.Type]bool{}
	}
	if v, ok := w.follows[t]; ok {
		return v
	}
	w.follows[t] = false
	value := false
	switch x := underlying(t).(type) {
	case *dwarf.PtrType:
		value = true
	case *dwarf.StructType:
		for _, f := range x.Field {
			if f.Type != nil && w.canFollow(f.Type) {
				value = true
				break
			}
		}
	case *dwarf.ArrayType:
		value = w.canFollow(x.Type)
	default:
		value = false
	}
	w.follows[t] = value
	return value
}

func (w *walker) add(addr, size uint64, root, path, kind string, iface bool) bool {
	s, ok := w.m.mapAddr(addr, size)
	if !ok {
		return false
	}
	c := candidate{span: s, Address: addr, Root: root, Path: path, Kind: kind}
	if iface {
		w.interfaces = append(w.interfaces, c)
	} else {
		w.static = append(w.static, c)
	}
	return true
}

func (w *walker) walk(t dwarf.Type, addr uint64, root, path string, depth int, via, iface bool) {
	if t == nil || depth > 24 || w.steps >= 250000 || t.Size() <= 0 || t.Size() > 16<<20 {
		return
	}
	if via {
		w.add(addr, uint64(t.Size()), root, path, "typed_object", iface)
	}
	if !w.canFollow(t) {
		return
	}
	key := visitKey{addr, t, iface}
	if w.seen[key] {
		w.cycles++
		return
	}
	w.seen[key] = true
	w.steps++
	if _, ok := w.m.read(addr, uint64(t.Size())); !ok {
		return
	}
	if w.kinds[t] == 20 || w.kinds[underlying(t)] == 20 {
		w.walkInterface(t, addr, root, path, depth)
		return
	}
	switch x := underlying(t).(type) {
	case *dwarf.PtrType:
		p, ok := w.m.ptr(addr)
		if ok && p != 0 {
			w.walk(x.Type, p, root, path+"*", depth+1, true, iface)
		}
	case *dwarf.StructType:
		if (x.StructName == "runtime.eface" || x.StructName == "runtime.iface") && x.Size() == 16 {
			w.walkInterface(t, addr, root, path, depth)
			return
		}
		if x.StructName == "string" || w.kinds[t] == 24 || w.kinds[x] == 24 {
			p, ok := w.m.ptr(addr)
			n, ok2 := w.m.ptr(addr + 8)
			if ok && ok2 && n <= 16<<20 {
				w.add(p, n, root, path+".bytes", "typed_string", iface)
			}
			return
		}
		if strings.HasPrefix(x.StructName, "[]") || w.kinds[t] == 23 || w.kinds[x] == 23 {
			if len(x.Field) != 3 {
				return
			}
			pt, ok := underlying(x.Field[0].Type).(*dwarf.PtrType)
			if !ok || pt.Type == nil || pt.Type.Size() <= 0 {
				return
			}
			p, a := w.m.ptr(addr)
			n, b := w.m.ptr(addr + 8)
			capacity, c := w.m.ptr(addr + 16)
			stride := uint64(pt.Type.Size())
			if !a || !b || !c || capacity < n || capacity > (16<<20)/stride {
				return
			}
			if !w.add(p, capacity*stride, root, path+".backing", "typed_slice", iface) {
				return
			}
			if !w.canFollow(pt.Type) {
				return
			}
			for i := uint64(0); i < min(n, 4096); i++ {
				w.walk(pt.Type, p+i*stride, root, fmt.Sprintf("%s[%d]", path, i), depth+1, false, iface)
			}
			return
		}
		for _, f := range x.Field {
			if f.Type != nil && f.ByteOffset >= 0 && f.Type.Size() >= 0 && f.ByteOffset+f.Type.Size() <= x.Size() {
				w.walk(f.Type, addr+uint64(f.ByteOffset), root, path+"."+f.Name, depth+1, false, iface)
			}
		}
	case *dwarf.ArrayType:
		if x.Type == nil || x.Type.Size() <= 0 || x.Count < 0 {
			return
		}
		for i := int64(0); i < min(x.Count, 4096); i++ {
			w.walk(x.Type, addr+uint64(i*x.Type.Size()), root, fmt.Sprintf("%s[%d]", path, i), depth+1, false, iface)
		}
	default:
		return
	}
}

func (w *walker) walkInterface(_ dwarf.Type, addr uint64, root, path string, depth int) {
	typeAddr, a := w.m.ptr(addr)
	data, b := w.m.ptr(addr + 8)
	if !a || !b || typeAddr == 0 || data == 0 {
		return
	}
	rt := w.runtimeTypes[typeAddr]
	if rt == nil { // A nonempty interface points to an itab; its second word is the concrete type.
		inter, ok := w.m.ptr(typeAddr)
		if !ok || w.runtimeTypes[inter] == nil || w.runtimeTypes[inter].Kind != reflect.Interface {
			return
		}
		if p, ok := w.m.ptr(typeAddr + 8); ok {
			typeAddr = p
			rt = w.runtimeTypes[p]
		}
	}
	if rt == nil {
		return
	}
	header, ok := w.m.read(typeAddr, 48)
	if !ok {
		return
	}
	size := binary.LittleEndian.Uint64(header)
	if size == 0 || size > 16<<20 {
		return
	}
	if header[23]&32 != 0 { // Direct-interface data is a value, not an address to a boxed value.
		if rt.Kind == reflect.Pointer && rt.Element != nil {
			if dt := w.types[rt.Element.String()]; dt != nil {
				w.walk(dt, data, root, path+".("+rt.String()+")", depth+1, true, true)
			}
		}
		return
	}
	if !w.add(data, size, root, path+".("+rt.String()+")", "runtime_validated_interface_box", true) {
		return
	}
	if dt := w.types[rt.String()]; dt != nil && dt.Size() == int64(size) {
		w.walk(dt, data, root, path+".("+rt.String()+")", depth+1, false, true)
	}
}

func scanStatic(m imageFile, g *gore.GoFile) (*walker, int, error) {
	w := &walker{m: m, seen: map[visitKey]bool{}, types: map[string]dwarf.Type{}, kinds: map[dwarf.Type]int64{}, runtimeTypes: map[uint64]*gore.GoType{}}
	d, err := m.file.DWARF()
	if err != nil {
		return w, 0, err
	}
	ts, err := g.GetTypes()
	if err == nil {
		for _, t := range ts {
			w.runtimeTypes[t.Addr] = t
		}
	}
	r := d.Reader()
	var roots []*dwarf.Entry
	for {
		e, err := r.Next()
		if err != nil {
			return w, 0, err
		}
		if e == nil {
			break
		}
		if off, ok := e.Val(dwarf.AttrType).(dwarf.Offset); ok {
			if t, err := d.Type(off); err == nil {
				w.types[t.String()] = t
			}
		}
		if k, ok := e.Val(dwarf.Attr(0x2900)).(int64); ok {
			if t, err := d.Type(e.Offset); err == nil {
				w.kinds[t] = k
				w.kinds[underlying(t)] = k
				w.types[t.String()] = t
			}
		}
		if e.Tag == dwarf.TagVariable {
			loc, ok := e.Val(dwarf.AttrLocation).([]byte)
			if ok && len(loc) == 9 && loc[0] == 3 {
				roots = append(roots, e)
			}
		}
	}
	for _, e := range roots {
		off, ok := e.Val(dwarf.AttrType).(dwarf.Offset)
		if !ok {
			continue
		}
		t, err := d.Type(off)
		if err != nil {
			continue
		}
		name, ok := e.Val(dwarf.AttrName).(string)
		if !ok {
			continue
		}
		loc, ok := e.Val(dwarf.AttrLocation).([]byte)
		if !ok || len(loc) != 9 || loc[0] != 3 {
			continue
		}
		w.walk(t, binary.LittleEndian.Uint64(loc[1:]), name, name, 0, false, false)
	}
	return w, len(roots), nil
}

func main() {
	out := flag.String("out", "", "report path")
	flag.Parse()
	if flag.NArg() != 1 {
		panic("usage: staticreach -out report.json binary")
	}
	path := flag.Arg(0)
	b, err := os.ReadFile(path)
	must(err)
	f, err := elf.Open(path)
	must(err)
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Machine != elf.EM_X86_64 || f.ByteOrder != binary.LittleEndian {
		panic("prototype supports little-endian ELF amd64 only")
	}
	if f.Type != elf.ET_EXEC {
		panic("prototype does not apply PIE/shared-object relocations; ET_EXEC required")
	}
	m := imageFile{f, b}
	input, err := os.Open(path)
	must(err)
	defer input.Close()
	base, err := internal.Analyze(path, input, uint64(len(b)), internal.Options{CoverageDetails: true})
	must(err)
	g, err := gore.Open(path)
	must(err)
	defer g.Close()
	w, roots, staticErr := scanStatic(m, g)
	r := report{Input: filepath.Base(path), FileSize: uint64(len(b)), Roots: roots, Cycles: w.cycles, Notes: []string{"Analysis only: input executable was never executed.", "Typed static references prove initialized file-backed reachability; they do not prove linker retention or deletion savings.", "Interface candidates additionally require runtime type matching, but remain separate from typed-reference totals.", "CFG strings are heuristic ABI register pairs validated by mapped printable UTF-8 bytes; their new bytes are candidates, not validated attribution.", "Prototype supports ELF64 little-endian amd64, static DW_OP_addr globals, bounded pointers/structs/arrays/slices, and a small x86 instruction subset.", "Shared/cyclic static objects are visited once globally; the representative root is not an exclusive owner.", "No performance comparisons: experiments may run concurrently."}}
	if staticErr != nil {
		r.Notes = append(r.Notes, "Static graph unavailable: "+staticErr.Error())
	}
	r.Notes = append(r.Notes, "No PIE/shared-object relocations, dynamic initializers, arbitrary unsafe pointers, or direct-interface arrays/structs are reconstructed. Interface direct values are followed only for pointer kinds.", "The four result families overlap and must not be added. New reference bytes split prior recognized structures from prior unclassified bytes.")
	if w.steps >= 250000 {
		r.Notes = append(r.Notes, "Static graph reached 250000-object budget")
	}
	r.Baseline.Attributed = base.Coverage.Attributed
	r.Baseline.Recognized = base.Coverage.Recognized
	r.Baseline.Unclassified = base.Coverage.Unclassified
	var attributed, recognized []span
	for _, s := range base.Coverage.Regions {
		switch s.Class {
		case "attributed":
			attributed = append(attributed, span{s.Offset, s.Size})
		case "recognized":
			recognized = append(recognized, span{s.Offset, s.Size})
		default:
			continue
		}
	}
	var cfg, cb []candidate
	seen := map[uint64]bool{}
	var visit func(entity.PackageMap)
	visit = func(ps entity.PackageMap) {
		for _, p := range ps {
			for _, file := range p.Files {
				for _, fn := range file.Functions {
					if seen[fn.Addr] {
						continue
					}
					seen[fn.Addr] = true
					r.Functions++
					for cr := range fn.CodeRegions {
						code, ok := m.read(cr.Addr, cr.Size)
						if !ok {
							continue
						}
						cs, budget := scanCFG(code, cr.Addr, p.Name+"."+fn.Name, m)
						if budget {
							r.SkippedFunctions++
						}
						cfg = append(cfg, cs...)
						for _, c := range cs {
							if c.CrossBlock {
								cb = append(cb, c)
							}
						}
					}
				}
			}
			visit(p.SubPackages)
		}
	}
	visit(base.Packages)
	r.Static = summarize(w.static, attributed, recognized)
	r.Interfaces = summarize(w.interfaces, attributed, recognized)
	r.CFG = summarize(cfg, attributed, recognized)
	r.CFGCrossBlock = summarize(cb, attributed, recognized)
	data, err := json.MarshalIndent(r, "", "  ")
	must(err)
	if *out == "" {
		fmt.Println(string(data))
	} else {
		must(os.MkdirAll(filepath.Dir(*out), 0o755))
		must(os.WriteFile(*out, append(data, '\n'), 0o600))
	}
	fmt.Fprintf(os.Stderr, "%s static=%d interface=%d cfg=%d cross-block=%d new bytes (CFG/interface are candidates)\n", r.Input, r.Static.New, r.Interfaces.New, r.CFG.New, r.CFGCrossBlock.New)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
