package entity

type AddrType = string

const (
	AddrTypeUnknown AddrType = "unknown" // it exists, but should never be collected
	AddrTypeText    AddrType = "text"    // for text section
	AddrTypeData    AddrType = "data"    // data / rodata section
)

type AddrSourceType = string

const (
	AddrRoleDefinition = "definition"
	AddrRoleReference  = "reference"
)

const (
	AddrSourceGoPclntab    AddrSourceType = "pclntab"
	AddrSourceSymbol       AddrSourceType = "symbol"
	AddrSourceDisasm       AddrSourceType = "disasm"
	AddrSourceDwarf        AddrSourceType = "dwarf"
	AddrSourceTyp          AddrSourceType = "type"
	AddrSourceStaticCopy   AddrSourceType = "static_copy"
	AddrSourceStaticRead   AddrSourceType = "static_read"
	AddrSourceStaticHeader AddrSourceType = "static_header"
)
