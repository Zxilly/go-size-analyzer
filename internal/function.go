package internal

type FuncType string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

type Function struct {
	Name     string
	Addr     uint64
	Size     uint64
	Type     FuncType
	Receiver string // only for methods
	Filepath string
	Pkg      *Package
}
