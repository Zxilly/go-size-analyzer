package main

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"flag"
	"fmt"
	"github.com/goretk/gore"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func getSourceDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return filepath.Dir(filename)
}

// found files to analyze:
func files() (ret []string) {
	pattern := "bin-"

	oss := strings.Split(*goos, ",")
	pattern += "(" + strings.Join(oss, "|") + ")"

	pattern += `-1.\d\d-amd64`

	if *ext {
		pattern += "(-ext)?"
	}
	pattern = "^" + pattern + "$"

	re := regexp.MustCompile(pattern)

	store := filepath.Join(getSourceDir(), "bins")

	entries, err := os.ReadDir(store)
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if re.MatchString(entry.Name()) {
			ret = append(ret, path.Join(store, entry.Name()))
		}
	}
	return
}

var ext = flag.Bool("ext", false, "analyze external link file")
var goos = flag.String("os", "linux", "target os")

func init() {
	flag.Parse()
}

func main() {
	for _, file := range files() {
		name := filepath.Base(file)

		gf, err := gore.Open(file)
		if err != nil {
			panic(err)
		}

		var symbolAddr uint64
		var textAddr uint64
		rf := gf.GetParsedFile()
		switch f := rf.(type) {
		case *pe.File:
			symbolAddr = peSymbolAddr(f)
			textAddr = peTextAddr(f)
		case *elf.File:
			symbolAddr = elfSymbolAddr(f)
			textAddr = elfTextAddr(f)
		case *macho.File:
			symbolAddr = machoSymbolAddr(f)
			textAddr = machoTextAddr(f)
		default:
			panic("This should not happened :(")
		}

		fmt.Printf("%s have offset %d\n", name, int64(symbolAddr)-int64(textAddr))

	}
}

func elfSymbolAddr(rf *elf.File) uint64 {
	symbols, err := rf.Symbols()
	if err != nil {
		panic(err)
	}
	for _, sym := range symbols {
		if sym.Name == "runtime.text" {
			return sym.Value
		}
	}
	panic("runtime.text not found")
}

func elfTextAddr(rf *elf.File) uint64 {
	for _, section := range rf.Sections {
		if section.Name == ".text" {
			return section.Addr
		}
	}
	panic(".text not found")
}

func machoSymbolAddr(rf *macho.File) uint64 {
	if rf.Symtab == nil {
		panic("rf.Symtab == nil")
	}
	for _, sym := range rf.Symtab.Syms {
		if sym.Name == "runtime.text" {
			return sym.Value
		}
	}
	panic("runtime.text not found")
}

func machoTextAddr(rf *macho.File) uint64 {
	for _, section := range rf.Sections {
		if section.Name == "__text" {
			return section.Addr
		}
	}
	panic("__text not found")
}

func peSymbolAddr(rf *pe.File) uint64 {
	for _, symb := range rf.Symbols {
		if symb.Name == "runtime.text" {
			return uint64(symb.Value)
		}
	}
	panic("main.main not found")
}

func peTextAddr(rf *pe.File) uint64 {
	for _, section := range rf.Sections {
		if section.Name == ".text" {
			return uint64(section.VirtualAddress)
		}
	}
	panic(".text not found")
}
