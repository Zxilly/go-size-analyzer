package entity

type FuncType = string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod            = "method"
)

type Function struct {
	Name     string   `json:"name"`
	Addr     uint64   `json:"addr"`
	Size     uint64   `json:"size"`
	Type     FuncType `json:"type"`
	Receiver string   `json:"receiver"` // only for methods
	File     *File    `json:"-"`

	disasm AddrSpace
	pkg    *Package
}
