package entity

type FuncType = string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

type Function struct {
	Name     string   `json:"name"`
	Addr     uint64   `json:"addr"`
	CodeSize uint64   `json:"code_size"`
	Type     FuncType `json:"type"`
	Receiver string   `json:"receiver"` // only for methods

	PclnSize PclnSymbolSize `json:"pcln_size"`

	File *File `json:"-"`

	disasm AddrSpace
	pkg    *Package
}

func (f *Function) Size() uint64 {
	return f.CodeSize + f.PclnSize.Size()
}
