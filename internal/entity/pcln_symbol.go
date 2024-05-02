package entity

import "github.com/ZxillyFork/gosym"

// PclnSymbolSize represents a pcln symbol sizes
type PclnSymbolSize struct {
	Name   uint64         `json:"name"`   // the function name size
	PCFile uint64         `json:"pcfile"` // the file name tab size
	PCSP   uint64         `json:"pcsp"`   // the pc to stack pointer table size
	PCLN   uint64         `json:"pcln"`   // the pc to line number table size
	Header uint64         `json:"header"` // the header size
	PCData map[string]int `json:"pcdata"` // the pcdata size
}

func (p *PclnSymbolSize) Size() uint64 {
	var size uint64
	size += p.Name
	size += p.PCFile
	size += p.PCSP
	size += p.PCLN
	size += p.Header
	for _, v := range p.PCData {
		size += uint64(v)
	}
	return size
}

func NewPclnSymbolSize(s *gosym.Func) *PclnSymbolSize {
	return &PclnSymbolSize{
		Name:   uint64(s.FuncNameSize()),
		PCFile: uint64(s.TablePCFileSize()),
		PCSP:   uint64(s.TablePCSPSize()),
		PCLN:   uint64(s.TablePCLnSize()),
		Header: uint64(s.FixedHeaderSize()),
		PCData: s.PCDataSize(),
	}
}
