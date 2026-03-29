package entity

type Analyzer = string

const (
	AnalyzerDwarf       Analyzer = "dwarf"
	AnalyzerDisasm      Analyzer = "disasm"
	AnalyzerSymbol      Analyzer = "symbol"
	AnalyzerPclntab     Analyzer = "pclntab"
	AnalyzerTyp         Analyzer = "type"
	AnalyzerPclntabMeta Analyzer = "pclntab_meta"
)
