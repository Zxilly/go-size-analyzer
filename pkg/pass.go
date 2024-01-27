package pkg

import "github.com/philpearl/intern"

var metaIntern = intern.New(4096)

type GoPclntabMeta struct {
	FuncName    string
	PackageName string
	Type        string // can be "function" or "method"
	Receiver    string // for method only
}

func (g GoPclntabMeta) GetInternedMeta() InternMeta {
	return GoPclntabMeta{
		FuncName:    metaIntern.Deduplicate(g.FuncName),
		PackageName: metaIntern.Deduplicate(g.PackageName),
		Type:        metaIntern.Deduplicate(g.Type),
		Receiver:    metaIntern.Deduplicate(g.Receiver),
	}
}

type SymbolMeta struct {
	SymbolName  string
	PackageName string
}

func (s SymbolMeta) GetInternedMeta() InternMeta {
	return SymbolMeta{
		SymbolName:  metaIntern.Deduplicate(s.SymbolName),
		PackageName: metaIntern.Deduplicate(s.PackageName),
	}
}

type DisasmMeta struct {
	GoPclntabMeta // disasm all functions from pclntab, so it has all info from pclntab
	DisasmIndex   int
	DisasmString  string
}

func (d DisasmMeta) GetInternedMeta() InternMeta {
	return DisasmMeta{
		GoPclntabMeta: d.GoPclntabMeta.GetInternedMeta().(GoPclntabMeta),
		DisasmIndex:   d.DisasmIndex,
		DisasmString:  metaIntern.Deduplicate(d.DisasmString),
	}
}

type InternMeta interface {
	// GetInternedMeta Intern the string in this meta
	GetInternedMeta() InternMeta
}

type AddrParsePass int

const (
	AddrPassGoPclntab AddrParsePass = iota
	AddrPassSymbol
	AddrPassDisasm
)

func (p AddrParsePass) String() string {
	switch p {
	case AddrPassGoPclntab:
		return "pclntab"
	case AddrPassSymbol:
		return "symbol"
	case AddrPassDisasm:
		return "disasm"
	default:
		return "unknown"
	}
}
