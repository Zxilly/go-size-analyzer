// Command report experiments with object evidence and function-level size diffs.
// It deliberately leaves the production report schema and byte ledger unchanged.
package main

import (
	"cmp"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

type manifest struct {
	Path              string            `json:"path"`
	SHA256            string            `json:"sha256"`
	GoVersion         string            `json:"go_version,omitempty"`
	MainPath          string            `json:"main_path,omitempty"`
	Settings          map[string]string `json:"settings,omitempty"`
	BuildInfoError    string            `json:"build_info_error,omitempty"`
	AnalyzerGoVersion string            `json:"analyzer_go_version"`
	AnalyzerSHA256    string            `json:"analyzer_sha256,omitempty"`
	AnalyzerRevision  string            `json:"analyzer_revision,omitempty"`
	AnalyzerModified  string            `json:"analyzer_modified,omitempty"`
	Modules           map[string]string `json:"modules,omitempty"`
}

type evidence struct {
	Source     string             `json:"source"`
	Role       string             `json:"role"`
	Boundary   string             `json:"boundary"`
	Address    uint64             `json:"address"`
	Size       uint64             `json:"size"`
	FileRanges []entity.FileRange `json:"file_ranges,omitempty"`
}

type object struct {
	ID             string     `json:"id"`
	Package        string     `json:"package"`
	File           string     `json:"file,omitempty"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	CodeBytes      uint64     `json:"code_bytes"`
	PclnBytes      uint64     `json:"pcln_bytes"`
	ReferenceBytes uint64     `json:"reference_bytes"`
	PhysicalBytes  uint64     `json:"physical_bytes"`
	EvidenceCount  int        `json:"evidence_count"`
	Sources        []string   `json:"sources"`
	Evidence       []evidence `json:"evidence_sample,omitempty"`
	ranges         []entity.FileRange
	refRanges      []entity.FileRange
	codeRanges     []entity.FileRange
	sourceSet      map[string]bool
}

type report struct {
	Version             int                  `json:"version"`
	Manifest            manifest             `json:"manifest"`
	Size                uint64               `json:"size"`
	Coverage            *entity.FileCoverage `json:"coverage"`
	Analyzers           []string             `json:"analyzers"`
	ObjectPhysicalUnion uint64               `json:"object_physical_union"`
	Objects             []*object            `json:"objects"`
	Notes               []string             `json:"notes"`
}

type change struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Package        string `json:"package"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	OldBytes       uint64 `json:"old_bytes"`
	NewBytes       uint64 `json:"new_bytes"`
	Delta          int64  `json:"delta"`
	CodeDelta      int64  `json:"code_delta"`
	PclnDelta      int64  `json:"pcln_delta"`
	ReferenceDelta int64  `json:"reference_delta"`
}

type comparison struct {
	Old              manifest `json:"old"`
	New              manifest `json:"new"`
	FileDelta        int64    `json:"file_delta"`
	Warnings         []string `json:"warnings"`
	Changed          []change `json:"changed"`
	UnchangedObjects int      `json:"unchanged_objects"`
}

func main() {
	if err := run(); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	out := flags.String("out", "", "JSON output (required)")
	html := flags.String("html", "", "optional standalone HTML evidence/diff explorer")
	oldPath := flags.String("old", "", "old experimental JSON report, for diff")
	newPath := flags.String("new", "", "new experimental JSON report, for diff")
	samples := flags.Int("evidence-samples", 3, "maximum stored evidence examples per object")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	if *oldPath != "" || *newPath != "" {
		oldReport, err := readReport(*oldPath)
		if err != nil {
			return err
		}
		newReport, err := readReport(*newPath)
		if err != nil {
			return err
		}
		diff := compare(oldReport, newReport)
		if err := writeJSON(*out, diff); err != nil {
			return err
		}
		if *html != "" {
			if err := writeHTML(*html, nil, diff); err != nil {
				return err
			}
		}
		fmt.Printf("changed=%d unchanged=%d file_delta=%d warnings=%d\n", len(diff.Changed), diff.UnchangedObjects, diff.FileDelta, len(diff.Warnings))
		return nil
	}
	if flags.NArg() != 1 {
		return errors.New("provide one binary, or -old and -new reports")
	}
	path := flags.Arg(0)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	r, err := internal.Analyze(path, f, uint64(st.Size()), internal.Options{})
	if err != nil {
		return err
	}
	rep := project(r, *samples)
	rep.Manifest, err = inspectManifest(path)
	if err != nil {
		return err
	}
	if err := writeJSON(*out, rep); err != nil {
		return err
	}
	if *html != "" {
		if err := writeHTML(*html, rep, nil); err != nil {
			return err
		}
	}
	counts := map[string]int{}
	withRefs := 0
	for _, o := range rep.Objects {
		counts[o.Kind]++
		if o.ReferenceBytes > 0 {
			withRefs++
		}
	}
	fmt.Printf("%s size=%d attributed=%d objects=%d kinds=%v objects_with_refs=%d object_union=%d\n", filepath.Base(path), rep.Size, rep.Coverage.Attributed, len(rep.Objects), counts, withRefs, rep.ObjectPhysicalUnion)
	return nil
}

func inspectManifest(path string) (manifest, error) {
	m := manifest{Path: path, AnalyzerGoVersion: runtime.Version(), Settings: map[string]string{}}
	if self, err := os.Executable(); err == nil {
		if b, err := os.ReadFile(self); err == nil {
			h := sha256.Sum256(b)
			m.AnalyzerSHA256 = hex.EncodeToString(h[:])
		}
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				m.AnalyzerRevision = s.Value
			}
			if s.Key == "vcs.modified" {
				m.AnalyzerModified = s.Value
			}
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return m, err
	}
	m.SHA256 = hex.EncodeToString(h.Sum(nil))
	b, err := buildinfo.ReadFile(path)
	if err != nil {
		m.BuildInfoError = err.Error()
		return m, nil
	}
	m.GoVersion, m.MainPath = b.GoVersion, b.Path
	m.Modules = map[string]string{}
	for _, dep := range b.Deps {
		value := dep.Version
		if dep.Replace != nil {
			value += " => " + dep.Replace.Path + "@" + dep.Replace.Version
		}
		m.Modules[dep.Path] = value
	}
	for _, setting := range b.Settings {
		m.Settings[setting.Key] = setting.Value
	}
	return m, nil
}

// IDs never contain a binary address. Receiver and generic spelling exposed by
// the production model are retained. That model may normalize linker names, so
// duplicate identities are aggregated rather than guessed to match by order.
func functionID(pkg string, f *entity.Function) string {
	return strings.Join([]string{"function", pkg, f.Type, f.Receiver, f.Name}, "\x1f")
}

func project(r *result.Result, sampleLimit int) *report {
	rep := &report{Version: 1, Size: r.Size, Coverage: r.Coverage, Analyzers: r.Analyzers}
	rep.Notes = []string{
		"Object sizes can overlap; their sum is not file size or removal savings. The production physical ledger remains authoritative.",
		"Evidence samples preserve existing function/symbol associations; instruction PC, full variable field paths and complete retention edges are not available in this API.",
		"Only native file-backed section mappings are projected. WASM object file ranges require its encoded-function/data mapping and are intentionally unavailable here.",
		"Pcln bytes and virtual code metrics are existing per-function measurements; physical_bytes is a separate file-range union.",
		"Symbols without stable names are excluded from cross-build matching; duplicate named identities are aggregated.",
		"Function identities use production-normalized names; generated names can collide. A production diff should retain original linker names and flag ambiguous matches.",
	}
	objects := map[string]*object{}
	functions := map[*entity.Function]*object{}
	get := func(id, pkg, name, kind, file string) *object {
		if o := objects[id]; o != nil {
			return o
		}
		o := &object{ID: id, Package: pkg, Name: name, Kind: kind, File: file, sourceSet: map[string]bool{}}
		objects[id] = o
		return o
	}
	var walk func(entity.PackageMap)
	walk = func(pkgs entity.PackageMap) {
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, fn := range file.Functions {
					o := get(functionID(pkg.Name, fn), pkg.Name, fn.Name, "function", file.FilePath)
					o.CodeBytes += fn.CodeSize
					o.PclnBytes += fn.PclnBytes()
					functions[fn] = o
				}
			}
			for claim := range pkg.OwnAddressValues {
				var o *object
				if claim.Function != nil {
					o = functions[claim.Function]
				}
				if o == nil && claim.Symbol != nil && claim.Symbol.Name != "" {
					s := claim.Symbol
					o = get(strings.Join([]string{"symbol", pkg.Name, s.Type, s.Name}, "\x1f"), pkg.Name, s.Name, "symbol", "")
				}
				if o == nil {
					continue
				}
				ranges := mapRanges(claim.Addr, claim.Size, r.Sections)
				if claim.Function != nil && claim.Function.Wasm {
					ranges = nil
				}
				o.ranges = append(o.ranges, ranges...)
				if o.Kind == "symbol" && claim.Type == entity.AddrTypeText {
					o.codeRanges = append(o.codeRanges, ranges...)
				}
				if claim.Role == entity.AddrRoleReference {
					o.refRanges = append(o.refRanges, ranges...)
				}
				o.EvidenceCount++
				o.sourceSet[claim.SourceType] = true
				if len(o.Evidence) < sampleLimit {
					o.Evidence = append(o.Evidence, evidence{Source: claim.SourceType, Role: claim.Role, Boundary: claim.Boundary, Address: claim.Addr, Size: claim.Size, FileRanges: ranges})
				}
			}
			walk(pkg.SubPackages)
		}
	}
	walk(r.Packages)
	var all []entity.FileRange
	for _, o := range objects {
		o.PhysicalBytes = entity.UnionFileSize(o.ranges)
		if o.Kind == "symbol" {
			o.CodeBytes = entity.UnionFileSize(o.codeRanges)
		}
		o.ReferenceBytes = entity.UnionFileSize(o.refRanges)
		all = append(all, o.ranges...)
		for source := range o.sourceSet {
			o.Sources = append(o.Sources, source)
		}
		slices.Sort(o.Sources)
		rep.Objects = append(rep.Objects, o)
	}
	rep.ObjectPhysicalUnion = entity.UnionFileSize(all)
	slices.SortFunc(rep.Objects, func(a, b *object) int {
		if n := cmp.Compare(b.PhysicalBytes, a.PhysicalBytes); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return rep
}

func mapRanges(addr, size uint64, sections []*entity.Section) []entity.FileRange {
	if size == 0 || addr+size < addr {
		return nil
	}
	var out []entity.FileRange
	for _, s := range sections {
		if s.OnlyInMemory || s.VirtualSection || s.Addr == 0 || s.FileSize == 0 {
			continue
		}
		length := min(s.Size, s.FileSize)
		if length > ^uint64(0)-s.Addr {
			continue
		}
		lo, hi := max(addr, s.Addr), min(addr+size, s.Addr+length)
		if lo < hi {
			out = append(out, entity.FileRange{Offset: s.Offset + lo - s.Addr, Size: hi - lo})
		}
	}
	return out
}

func readReport(path string) (*report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r report
	err = json.NewDecoder(f).Decode(&r)
	if err == nil && r.Version != 1 {
		err = fmt.Errorf("unsupported experimental report version %d", r.Version)
	}
	return &r, err
}

func compare(old, next *report) *comparison {
	c := &comparison{Old: old.Manifest, New: next.Manifest, FileDelta: int64(next.Size) - int64(old.Size)}
	if old.Manifest.GoVersion != next.Manifest.GoVersion {
		c.Warnings = append(c.Warnings, "Target Go version changed")
	}
	for _, key := range []string{"GOOS", "GOARCH", "CGO_ENABLED", "-buildmode", "-compiler", "-tags", "-gcflags", "-ldflags", "-trimpath", "GOAMD64", "GOARM64"} {
		if old.Manifest.Settings[key] != next.Manifest.Settings[key] {
			c.Warnings = append(c.Warnings, "Build setting changed: "+key)
		}
	}
	if old.Manifest.BuildInfoError != "" || next.Manifest.BuildInfoError != "" {
		c.Warnings = append(c.Warnings, "Build information unavailable; comparability cannot be established")
	}
	if old.Manifest.AnalyzerGoVersion != next.Manifest.AnalyzerGoVersion {
		c.Warnings = append(c.Warnings, "Analyzer Go version changed")
	}
	if old.Manifest.AnalyzerSHA256 != next.Manifest.AnalyzerSHA256 {
		c.Warnings = append(c.Warnings, "Analyzer executable changed")
	}
	if strings.Join(old.Analyzers, ",") != strings.Join(next.Analyzers, ",") {
		c.Warnings = append(c.Warnings, "Analyzer availability changed")
	}
	if old.Coverage != nil && next.Coverage != nil && old.Coverage.Version != next.Coverage.Version {
		c.Warnings = append(c.Warnings, "Coverage accounting version changed")
	}
	c.Warnings = append(c.Warnings, "Object deltas overlap and must not be summed to reproduce the file delta; no deletion savings are inferred")
	previous := map[string]*object{}
	for _, o := range old.Objects {
		previous[o.ID] = o
	}
	for _, n := range next.Objects {
		o := previous[n.ID]
		delete(previous, n.ID)
		ch := change{ID: n.ID, Name: n.Name, Package: n.Package, Kind: n.Kind, Status: "added", NewBytes: n.PhysicalBytes}
		if o != nil {
			ch.Status = "changed"
			ch.OldBytes = o.PhysicalBytes
		} else {
			o = &object{}
		}
		ch.Delta = int64(n.PhysicalBytes) - int64(o.PhysicalBytes)
		ch.CodeDelta = int64(n.CodeBytes) - int64(o.CodeBytes)
		ch.PclnDelta = int64(n.PclnBytes) - int64(o.PclnBytes)
		ch.ReferenceDelta = int64(n.ReferenceBytes) - int64(o.ReferenceBytes)
		if ch.Status == "changed" && ch.Delta == 0 && ch.CodeDelta == 0 && ch.PclnDelta == 0 && ch.ReferenceDelta == 0 {
			c.UnchangedObjects++
			continue
		}
		c.Changed = append(c.Changed, ch)
	}
	for _, o := range previous {
		c.Changed = append(c.Changed, change{ID: o.ID, Name: o.Name, Package: o.Package, Kind: o.Kind, Status: "removed", OldBytes: o.PhysicalBytes, Delta: -int64(o.PhysicalBytes), CodeDelta: -int64(o.CodeBytes), PclnDelta: -int64(o.PclnBytes), ReferenceDelta: -int64(o.ReferenceBytes)})
	}
	abs := func(n int64) uint64 {
		if n < 0 {
			return uint64(-n)
		}
		return uint64(n)
	}
	slices.SortFunc(c.Changed, func(a, b change) int {
		if n := cmp.Compare(abs(b.Delta), abs(a.Delta)); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return c
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(value)
}

func writeHTML(path string, r *report, c *comparison) error {
	data := struct {
		Report  *report
		Diff    *comparison
		Rows    []*object
		Changes []change
	}{Report: r, Diff: c}
	if r != nil {
		data.Rows = r.Objects[:min(300, len(r.Objects))]
	}
	if c != nil {
		data.Changes = c.Changed[:min(300, len(c.Changed))]
	}
	t, err := template.New("report").Parse(page)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

const page = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Binary size evidence experiment</title>
<style>body{font:15px system-ui;margin:40px auto;max-width:1280px;padding:0 24px;color:#172233;background:#f7f8fa}h1{font-size:30px}input{padding:12px;width:65%;border:1px solid #aab5c2;border-radius:5px}table{border-collapse:collapse;width:100%;background:white;margin:24px 0}th,td{text-align:left;vertical-align:top;padding:10px;border-bottom:1px solid #dde2e8}th{position:sticky;top:0;background:#e6edf6}td:first-child{max-width:550px;overflow-wrap:anywhere}small{color:#546171}pre{white-space:pre-wrap;font-size:12px}.metrics{font-size:18px}summary{cursor:pointer}.warning{border-left:3px solid #ad7020;padding-left:14px}</style>
<h1>Binary size evidence experiment</h1><p>Physical file accounting, object evidence and cross-build changes. Object sizes may overlap; they do not represent deletion savings.</p>
{{if .Report}}<p class="metrics">{{.Report.Manifest.Path}} · {{.Report.Size}} file bytes · {{len .Report.Objects}} objects</p><p>Attributed {{.Report.Coverage.Attributed}} · Recognized {{.Report.Coverage.Recognized}} · Unclassified {{.Report.Coverage.Unclassified}}</p><p>Go {{.Report.Manifest.GoVersion}} · SHA256 {{.Report.Manifest.SHA256}}</p>{{range .Report.Notes}}<p class="warning">{{.}}</p>{{end}}{{end}}
{{if .Diff}}<p class="metrics">File change: {{.Diff.FileDelta}} bytes · {{len .Diff.Changed}} changed objects · {{.Diff.UnchangedObjects}} unchanged</p>{{range .Diff.Warnings}}<p class="warning">{{.}}</p>{{end}}{{end}}
<label>Filter displayed objects <input id="filter" placeholder="Package, function or symbol"></label><p id="count"></p><small>Preview shows up to 300 largest objects/changes. The JSON contains every object. Evidence is a bounded sample, not a complete reference graph.</small>
{{if .Report}}<table><thead><tr><th>Object and evidence</th><th>Physical union</th><th>Code</th><th>Pcln</th><th>Referenced data</th></tr></thead><tbody>{{range .Rows}}<tr><td><details><summary>{{.Name}}</summary><p>{{.Package}} · {{.Kind}}</p><p>{{.File}}</p><p>{{.EvidenceCount}} claims · {{.Sources}}</p>{{range .Evidence}}<pre>{{.Source}} / {{.Role}} / {{.Boundary}}
Address {{.Address}} · {{.Size}} bytes · File ranges {{.FileRanges}}</pre>{{end}}</details></td><td>{{.PhysicalBytes}}</td><td>{{.CodeBytes}}</td><td>{{.PclnBytes}}</td><td>{{.ReferenceBytes}}</td></tr>{{end}}</tbody></table>{{end}}
{{if .Diff}}<table><thead><tr><th>Object</th><th>Status</th><th>Old</th><th>New</th><th>Physical Δ</th><th>Code Δ</th><th>Pcln Δ</th><th>Reference Δ</th></tr></thead><tbody>{{range .Changes}}<tr><td>{{.Name}}<br><small>{{.Package}}</small></td><td>{{.Status}}</td><td>{{.OldBytes}}</td><td>{{.NewBytes}}</td><td>{{.Delta}}</td><td>{{.CodeDelta}}</td><td>{{.PclnDelta}}</td><td>{{.ReferenceDelta}}</td></tr>{{end}}</tbody></table>{{end}}
<script>const filter=document.getElementById('filter');function update(){let n=0;for(const row of document.querySelectorAll('tbody tr')){row.hidden=!row.textContent.toLowerCase().includes(filter.value.toLowerCase());if(!row.hidden)n++;}document.getElementById('count').textContent=n+' displayed rows';}filter.addEventListener('input',update);update();</script></html>`
