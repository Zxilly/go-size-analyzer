package internal

type FuncType = string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod            = "method"
)

type Function struct {
	Name     string    `json:"name"`
	Addr     uint64    `json:"addr"`
	Size     uint64    `json:"size"`
	Type     FuncType  `json:"type"`
	Receiver string    `json:"receiver"` // only for methods
	Filepath string    `json:"filepath"`
	Disasm   AddrSpace `json:"-"`
	Pkg      *Package  `json:"-"`
}
