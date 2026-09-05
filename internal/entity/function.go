package entity

type FuncType = string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

type Function struct {
	Name     string    `json:"name"`
	Addr     uint64    `json:"addr"`
	CodeSize uint64    `json:"code_size"`
	Type     FuncType  `json:"type"`
	Receiver string    `json:"receiver"`         // only for methods
	Ranges   []AddrPos `json:"ranges,omitempty"` // non-contiguous DWARF code ranges

	PclnSize PclnSymbolSize `json:"pcln_size"`

	disasm AddrSpace
	pkg    *Package
}

func (f *Function) CodeRegions(yield func(AddrPos) bool) {
	if len(f.Ranges) == 0 {
		yield(AddrPos{Addr: f.Addr, Size: f.CodeSize, Type: AddrTypeText})
		return
	}
	for _, r := range f.Ranges {
		if !yield(r) {
			return
		}
	}
}

func (f *Function) Init() {
	f.disasm = make(AddrSpace)
}

func (f *Function) Size() uint64 {
	return f.CodeSize + f.PclnSize.Size()
}
