package main

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"flag"
	"fmt"
	"github.com/goretk/gore"
	"log"
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

var ext = flag.Bool("ext", true, "analyze external link file")
var pie = flag.Bool("pie", true, "analyze pie file")
var goos = flag.String("os", "linux,darwin,windows", "target os")

func init() {
	flag.Parse()
}

func main() {
	for _, file := range files() {
		name := filepath.Base(file)

		fmt.Printf("%-30s: ", name)

		gf, err := gore.Open(file)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}

		rf := gf.GetParsedFile()
		switch f := rf.(type) {
		case *pe.File:
			printSegNamePE(f)
		case *elf.File:
			printSegNameELF(f)
		case *macho.File:
			printSegNameMACHO(f)
		default:
			panic("This should not happened :(")
		}
		fmt.Println()
	}
}

const symName = "runtime.pclntab"

func printSegNamePE(f *pe.File) {
	for _, sym := range f.Symbols {
		if sym.Name == symName {
			sect := f.Sections[sym.SectionNumber-1]
			fmt.Printf("PE: %s", sect.Name)
			return
		}
	}
}

func printSegNameELF(f *elf.File) {
	symbols, err := f.Symbols()
	if err != nil {
		panic(err)
	}

	for _, sym := range symbols {
		if sym.Name == symName {
			sect := f.Sections[sym.Section]
			fmt.Printf("ELF: %s", sect.Name)
			return
		}
	}
}

func printSegNameMACHO(f *macho.File) {
	for _, sym := range f.Symtab.Syms {
		if sym.Name == symName {
			sect := f.Sections[sym.Sect-1]
			fmt.Printf("MACHO: %s", sect.Name)
			return
		}
	}
}
