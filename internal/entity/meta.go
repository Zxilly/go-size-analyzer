package entity

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
	Value string
}

type DwarfMeta struct {
}
