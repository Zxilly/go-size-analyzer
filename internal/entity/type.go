package entity

type AddrType string

const (
	AddrTypeUnknown AddrType = "unknown" // it exists, but should never be collected
	AddrTypeText             = "text"    // for text section
	AddrTypeData             = "data"    // data / rodata section
)

type AddrSourceType = string

const (
	AddrSourceGoPclntab AddrSourceType = "pclntab"
	AddrSourceSymbol                   = "symbol"
	AddrSourceDisasm                   = "disasm"
)
