package main

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/goretk/gore"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
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
	if *pie {
		pattern += "(-pie)?"
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
var pie = flag.Bool("pie", false, "analyze pie file")
var goos = flag.String("os", "linux", "target os")

// currently have no way to analyze:
var ignores = []string{
	"bin-linux-1.21-amd64-ext-pie",
}

func init() {
	flag.Parse()
}

func main() {
	fmt.Printf("%-26s %10s %10s %10s %10s %10s %10s %10s\n",
		"Name",
		"sAddr", "mdAddr", "tAddr", "pe ibase",
		"sbased", "mbased", "tbased")

	for _, file := range files() {
		name := filepath.Base(file)

		if slices.Contains(ignores, name) {
			continue
		}

		gf, err := gore.Open(file)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}

		var symbolAddr uint64
		var textAddr uint64
		var moduleDataAddr uint64

		md, err := gf.Moduledata()
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		moduleDataAddr = md.Text().Address

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

		const target = "UsingConstString"

		pkgs, err := gf.GetPackages()
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}

		targetOffset := uint64(0)

		for _, pkg := range pkgs {
			for _, f := range pkg.Functions {
				if f.Name == target {
					targetOffset = f.Offset
					break
				}
			}
		}
		if targetOffset == 0 {
			log.Fatalf("%s: %v", name, "target function not found")
		}

		targetOffset -= textAddr
		if _, ok := rf.(*pe.File); ok {
			targetOffset -= peImageBase(rf.(*pe.File))
		}

		name = name[4:]

		fmt.Printf("%-26s %10x %s %s", name, symbolAddr, colored(symbolAddr, moduleDataAddr), colored(symbolAddr, textAddr))
		if _, ok := rf.(*pe.File); ok {
			fmt.Printf(" %10x", peImageBase(rf.(*pe.File)))
		} else {
			fmt.Printf(" %10s", "")
		}
		fmt.Printf(" %10x %10x %10x", targetOffset+symbolAddr, targetOffset+moduleDataAddr, targetOffset+textAddr)
		fmt.Println()
	}
}

func colored(real, check uint64) string {
	if real == check {
		return color.GreenString("%10x", check)
	} else {
		return color.RedString("%10x", check)
	}
}

func peImageBase(rf *pe.File) uint64 {
	switch hdr := rf.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return uint64(hdr.ImageBase)
	case *pe.OptionalHeader64:
		return hdr.ImageBase
	}
	panic("unknown optional header")
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
