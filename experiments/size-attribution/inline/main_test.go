package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNestedSourceProjection(t *testing.T) {
	items := []occurrence{
		{Origin: "outer", Host: "host", Depth: 1, FileRanges: []span{{10, 30}}},
		{Origin: "inner", Host: "host", Depth: 2, FileRanges: []span{{15, 20}, {18, 25}}},
		{Origin: "ambiguous", Host: "host", Depth: 2, FileRanges: []span{{23, 27}}},
	}
	groups, total, ambiguous := summarizeSources(items)
	if total != 20 || ambiguous != 2 {
		t.Fatalf("total=%d ambiguous=%d", total, ambiguous)
	}
	var sum uint64
	for _, g := range groups {
		sum += g.DeepestBytes
	}
	if sum+ambiguous != total {
		t.Fatalf("projection does not conserve: %d + %d != %d", sum, ambiguous, total)
	}
}

func TestGenericGrouping(t *testing.T) {
	for _, name := range []string{"p.F[go.shape.int]", "p.F[[2]byte]"} {
		d, _, ok := genericDeclaration(name)
		if !ok || d != "p.F" {
			t.Fatalf("%q => %q, %v", name, d, ok)
		}
	}
	for _, name := range []string{"p..dict.F[int]", "p.F[bad", "p.F]bad[", "type:.eq.[4]int", "type..eq.[4]int"} {
		if _, _, ok := genericDeclaration(name); ok {
			t.Fatalf("must not group %q", name)
		}
	}
}

func TestCompiledFixtureELFAndPE(t *testing.T) {
	if testing.Short() {
		t.Skip("requires cross-compilation")
	}
	for _, target := range []string{"linux", "windows"} {
		t.Run(target, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fixture.bin")
			cmd := exec.Command("go", "build", "-mod=readonly", "-p=2", "-o", path, "./testdata/fixture.go")
			cmd.Env = append(os.Environ(), "GOOS="+target, "GOARCH=amd64", "CGO_ENABLED=0", "GOMAXPROCS=4")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			r, err := analyze(path)
			if err != nil {
				t.Fatal(err)
			}
			if !r.DWARFAvailable {
				t.Fatalf("DWARF missing: %v", r.Notes)
			}
			found := false
			for _, in := range r.Inlines {
				if in.Host == "main.host" && in.Name == "main.addOne" {
					if size(in.FileRanges) == 0 {
						t.Fatal("inline has no mapped bytes")
					}
					found = true
				}
			}
			if !found {
				t.Fatal("expected addOne source bytes inside main.host")
			}
			found = false
			for _, g := range r.GenericGroups {
				if g.Declaration == "main.genericFirst" {
					if len(g.Bodies) != 2 {
						t.Fatalf("expected two shape bodies, got %d", len(g.Bodies))
					}
					for _, b := range g.Bodies {
						if b.Kind != "shared-shape-body" || b.Bytes == 0 || !strings.Contains(b.Name, "go.shape.") {
							t.Fatalf("invalid body: %+v", b)
						}
					}
					found = true
				}
			}
			if !found {
				t.Fatal("missing genericFirst declaration")
			}
			dictionaries := 0
			for _, name := range r.DictionarySymbols {
				if strings.HasPrefix(name, "main..dict.genericFirst[") {
					dictionaries++
				}
			}
			if dictionaries != 3 {
				t.Fatalf("three source type arguments should retain three dictionaries and two shared shape bodies, got %d dictionaries", dictionaries)
			}
			var deepest uint64
			for _, g := range r.Sources {
				deepest += g.DeepestBytes
			}
			if deepest+r.DeepestAmbiguousBytes != r.InlineUnionBytes {
				t.Fatal("source projection does not conserve")
			}
			for _, in := range r.Inlines {
				for _, s := range in.FileRanges {
					if s.End > uint64(r.FileSize) || s.Start >= s.End {
						t.Fatal("out-of-bounds span")
					}
				}
			}
		})
	}
}

func TestELFSymbolFallbackWithoutDWARF(t *testing.T) {
	if testing.Short() {
		t.Skip("requires cross-compilation")
	}
	path := filepath.Join(t.TempDir(), "fixture-no-dwarf")
	cmd := exec.Command("go", "build", "-mod=readonly", "-p=2", "-ldflags=-w", "-o", path, "./testdata/fixture.go")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOMAXPROCS=4")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	r, err := analyze(path)
	if err != nil {
		t.Fatal(err)
	}
	if r.DWARFAvailable || r.InlineOccurrences != 0 || r.GenericSource != "native_symbol" {
		t.Fatalf("invalid fallback: dwarf=%v inline=%d source=%s", r.DWARFAvailable, r.InlineOccurrences, r.GenericSource)
	}
	for _, g := range r.GenericGroups {
		if g.Declaration == "main.genericFirst" {
			if len(g.Bodies) != 2 || g.Bytes == 0 {
				t.Fatalf("unexpected bodies: %+v", g)
			}
			for _, b := range g.Bodies {
				if b.Source != "native_symbol" {
					t.Fatalf("wrong evidence source: %+v", b)
				}
			}
			return
		}
	}
	t.Fatal("missing native-symbol genericFirst group")
}
