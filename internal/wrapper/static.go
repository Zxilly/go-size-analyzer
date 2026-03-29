package wrapper

import (
	"errors"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

// src\cmd\link\internal\ld\data.go
var ignoreSymbols = utils.NewSet[string]()

// PclntabSymbols are runtime symbols whose addresses we capture for pclntab sub-table analysis.
var PclntabSymbols = utils.NewSet[string]()

func init() {
	symbols := []string{
		"runtime.text",
		"runtime.etext",
		"runtime.rodata",
		"runtime.erodata",
		"runtime.noptrdata",
		"runtime.enoptrdata",
		"runtime.bss",
		"runtime.ebss",
		"runtime.gcdata",
		"runtime.gcbss",
		"runtime.noptrbss",
		"runtime.enoptrbss",
		"runtime.end",
		"runtime.covctrs",
		"runtime.ecovctrs",

		"runtime.__start___sancov_cntrs",
		"runtime.__stop___sancov_cntrs",
		"internal/fuzz._counters",
		"internal/fuzz._ecounters",

		"runtime.types",
		"runtime.etypes",

		"runtime.itablink",
		"runtime.symtab",
		"runtime.esymtab",
		"runtime.pclntab",
		"runtime.pcheader",

		"runtime.zerobase",

		"go:buildinfo",
		"go:buildinfo.ref",
	}

	for _, sym := range symbols {
		ignoreSymbols.Add(sym)
	}

	pclntabSyms := []string{
		"runtime.funcnametab",
		"runtime.cutab",
		"runtime.filetab",
		"runtime.pctab",
		"runtime.functab",
		"runtime.epclntab",
	}

	for _, sym := range pclntabSyms {
		PclntabSymbols.Add(sym)
		ignoreSymbols.Add(sym)
	}
}

var ErrAddrNotFound = errors.New("address not found")
