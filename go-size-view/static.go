package go_size_view

import (
	mapset "github.com/deckarep/golang-set/v2"
)

// src\cmd\link\internal\ld\data.go
var ignoreSymbols = mapset.NewSet[string]()

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

		"runtime.rodata",
		"runtime.erodata",
		"runtime.types",
		"runtime.etypes",

		"runtime.itablink",
		"runtime.symtab",
		"runtime.esymtab",
		"runtime.pclntab",
		"runtime.pcheader",
		"runtime.funcnametab",
		"runtime.cutab",
		"runtime.filetab",
		"runtime.pctab",
		"runtime.functab",
		"runtime.epclntab",

		"runtime.zerobase",

		"go:buildinfo",
		"go:buildinfo.ref",
	}

	for _, sym := range symbols {
		ignoreSymbols.Add(sym)
	}
}
