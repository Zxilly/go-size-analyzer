package main

import (
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ZxillyFork/gore"
	"golang.org/x/arch/x86/x86asm"
)

func synthetic(data []byte) imageFile {
	return imageFile{file: &elf.File{Progs: []*elf.Prog{{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Vaddr: 0x1000, Off: 0, Filesz: uint64(len(data))}}}}, data: data}
}

func TestSubtractPhysicalUnion(t *testing.T) {
	a := []span{{10, 10}, {15, 20}, {100, 4}}
	b := []span{{0, 12}, {18, 10}, {102, 8}}
	got := subtract(a, b)
	want := []span{{12, 6}, {28, 7}, {100, 2}}
	if len(got) != len(want) {
		t.Fatal(got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatal(got)
		}
	}
}

func TestTypedPointerCycleSharedAndFileBounds(t *testing.T) {
	b := make([]byte, 128)
	binary.LittleEndian.PutUint64(b[0:], 0x1020)
	binary.LittleEndian.PutUint64(b[32:], 0x1040)
	binary.LittleEndian.PutUint64(b[64:], 0x1020)
	node := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 8}, StructName: "Node", Kind: "struct"}
	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: node}
	node.Field = []*dwarf.StructField{{Name: "Next", Type: ptr}}
	w := &walker{m: synthetic(b), seen: map[visitKey]bool{}, kinds: map[dwarf.Type]int64{}}
	w.walk(ptr, 0x1000, "root", "root", 0, false, false)
	if w.cycles == 0 {
		t.Fatal("cycle not detected")
	}
	if got := summarize(w.static, nil).Unique; got != 16 {
		t.Fatalf("got %d bytes, want 16", got)
	}
	w.walk(ptr, 0x1000, "shared", "shared", 0, false, false)
	if got := summarize(w.static, nil).Unique; got != 16 {
		t.Fatal(got)
	}
	binary.LittleEndian.PutUint64(b[0:], 0x2000)
	w.seen = map[visitKey]bool{}
	w.static = nil
	w.walk(ptr, 0x1000, "unmapped", "unmapped", 0, false, false)
	if len(w.static) != 0 {
		t.Fatal("unmapped pointer attributed")
	}
}

func TestCFGCarriesAddressAcrossBranch(t *testing.T) {
	// LEA 0x1040(%RIP),RAX; TEST RCX,RCX; JZ join; NOP; join: MOV $8,RBX; RET.
	code := []byte{0x48, 0x8d, 0x05, 0x39, 0, 0, 0, 0x48, 0x85, 0xc9, 0x74, 0x01, 0x90, 0x48, 0xc7, 0xc3, 0x08, 0, 0, 0, 0xc3}
	b := make([]byte, 128)
	copy(b, code)
	copy(b[64:], "CFG_TEXT")
	cs, budget := scanCFG(code, 0x1000, "fixture", synthetic(b))
	if budget || len(cs) != 1 || !cs[0].CrossBlock || cs[0].Offset != 64 || cs[0].Size != 8 {
		t.Fatalf("%+v budget %v", cs, budget)
	}
}

func TestCFGRejectsConflictingJoin(t *testing.T) {
	// Conditional path overwrites RAX; the join must discard the address.
	code := []byte{0x48, 0x8d, 0x05, 0x39, 0, 0, 0, 0x48, 0x85, 0xc9, 0x74, 0x07, 0x48, 0xc7, 0xc0, 0x00, 0x10, 0, 0, 0x48, 0xc7, 0xc3, 0x08, 0, 0, 0, 0xc3}
	b := make([]byte, 128)
	copy(b, code)
	copy(b[64:], "CFG_TEXT")
	cs, _ := scanCFG(code, 0x1000, "conflict", synthetic(b))
	if len(cs) != 0 {
		t.Fatalf("conflicting branch produced %+v", cs)
	}
}

func TestCFGCallClobbersFacts(t *testing.T) {
	code := []byte{0x48, 0x8d, 0x05, 0x39, 0, 0, 0, 0xe8, 0, 0, 0, 0, 0x48, 0xc7, 0xc3, 8, 0, 0, 0, 0xc3}
	b := make([]byte, 128)
	copy(b, code)
	copy(b[64:], "CFG_TEXT")
	cs, _ := scanCFG(code, 0x1000, "clobber", synthetic(b))
	if len(cs) != 0 {
		t.Fatal(cs)
	}
}

func TestCompiledStaticPointerAndInterface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.elf")
	cmd := exec.Command("go", "build", "-mod=readonly", "-p=2", "-o", path, "./fixture")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOMAXPROCS=4")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, output)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	w, _, err := scanStatic(imageFile{f, b}, g)
	if err != nil {
		t.Fatal(err)
	}
	pointerFound, interfaceFound := false, false
	for _, c := range w.static {
		if strings.HasPrefix(c.Path, "main.Root*") && c.Kind == "typed_string" {
			pointerFound = true
		}
	}
	for _, c := range w.interfaces {
		if c.Root == "main.Box" && c.Kind == "runtime_validated_interface_box" {
			interfaceFound = true
		}
	}
	if !pointerFound || !interfaceFound {
		t.Fatalf("pointer=%v interface=%v, interfaces=%+v", pointerFound, interfaceFound, w.interfaces)
	}
}

func TestCFGPartialAndImplicitWritesDiscardFacts(t *testing.T) {
	for name, instructionBytes := range map[string][]byte{"AL": {0xb0, 1}, "AX": {0x66, 0xb8, 1, 0}, "XCHG": {0x48, 0x87, 0xc1}, "MUL": {0x48, 0xf7, 0xe1}} {
		t.Run(name, func(t *testing.T) {
			inst, err := x86asm.Decode(instructionBytes, 64)
			if err != nil {
				t.Fatal(err)
			}
			var s state
			s[register(x86asm.RAX)] = fact{kind: 2, value: 0x1040}
			s[register(x86asm.RCX)] = fact{kind: 2, value: 0x1080}
			got := transfer(instruction{inst: inst}, s)
			if got != (state{}) {
				t.Fatalf("stale state: %+v", got)
			}
		})
	}
}

func TestCFG32BitLEAZeroExtends(t *testing.T) {
	inst, err := x86asm.Decode([]byte{0x8d, 0x40, 0x01}, 64)
	if err != nil {
		t.Fatal(err)
	}
	var s state
	s[register(x86asm.RAX)] = fact{kind: 2, value: 0x100001000}
	got := transfer(instruction{inst: inst}, s)
	if got[register(x86asm.RAX)].value != 0x1001 {
		t.Fatal(got)
	}
}

func TestDirectInterfacePointerArrayIsNotDoubleDereferenced(t *testing.T) {
	b := make([]byte, 256)
	binary.LittleEndian.PutUint64(b, 0x1040)
	binary.LittleEndian.PutUint64(b[8:], 0x1080)
	binary.LittleEndian.PutUint64(b[64:], 8)
	b[64+23] = 32 | 17
	array := &gore.GoType{Kind: reflect.Array, Element: &gore.GoType{Kind: reflect.Pointer, Name: "*Node"}}
	w := &walker{m: synthetic(b), runtimeTypes: map[uint64]*gore.GoType{0x1040: array}, types: map[string]dwarf.Type{}}
	w.walkInterface(nil, 0x1000, "array", "array", 0)
	if len(w.interfaces) != 0 {
		t.Fatal("unsupported direct array followed as pointer")
	}
}

func TestSummarySeparatesRecognizedBytes(t *testing.T) {
	s := summarize([]candidate{{span: span{10, 20}}}, []span{{10, 5}}, []span{{15, 8}})
	if s.New != 15 || s.NewRecognized != 8 || s.NewUnclassified != 7 {
		t.Fatal(s)
	}
}
