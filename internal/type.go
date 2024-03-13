package internal

import "go4.org/intern"

func Deduplicate(s string) string {
	return intern.GetByString(s).Get().(string)
}

type GoPclntabMeta struct {
	FuncName    string
	PackageName string
	Type        FuncType
	Receiver    string // for method only
	Filepath    string
}

type SymbolMeta struct {
	SymbolName  string
	PackageName string
}

type DisasmMeta struct {
	Source      GoPclntabMeta // disasm all functions from pclntab, so it has all info from pclntab
	DisasmIndex int
}

type AddrSourceType string

const (
	AddrSourceGoPclntab AddrSourceType = "pclntab"
	AddrSourceSymbol                   = "symbol"
	AddrSourceDisasm                   = "disasm"
)
