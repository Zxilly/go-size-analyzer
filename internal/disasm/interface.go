package disasm

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"unicode/utf8"
)

type PossibleStr struct {
	Addr uint64
	Size uint64
}

type extractorFunc func(code []byte, pc uint64) []PossibleStr

type Extractor struct {
	raw       wrapper.RawFileWrapper
	size      uint64
	text      []byte        // bytes of text segment (actual instructions)
	textStart uint64        // start PC of text
	textEnd   uint64        // end PC of text
	goarch    string        // GOARCH string
	extractor extractorFunc // disassembler function for goarch
}

func NewExtractor(rawFile wrapper.RawFileWrapper, size uint64) (*Extractor, error) {
	textStart, text, err := rawFile.Text()
	if err != nil {
		return nil, err
	}

	goarch := rawFile.GoArch()
	if goarch == "" {
		return nil, fmt.Errorf("unknown GOARCH")
	}
	extractFunc := extractFuncs[goarch]
	if extractFunc == nil {
		return nil, fmt.Errorf("unsupported GOARCH %s", goarch)
	}

	return &Extractor{
		raw:       rawFile,
		size:      size,
		text:      text,
		textStart: textStart,
		textEnd:   textStart + uint64(len(text)),
		goarch:    goarch,
		extractor: extractFunc,
	}, nil
}

func (e *Extractor) Extract(start, end uint64) []PossibleStr {
	if start < e.textStart {
		panic(fmt.Sprintf("start address %#x is before text segment %#x", start, e.textStart))
	}
	if end > e.textEnd {
		panic(fmt.Sprintf("end address %#x is after text segment %#x", end, e.textEnd))
	}

	code := e.text[start-e.textStart : end-e.textStart]

	return e.extractor(code, start)
}

func (e *Extractor) AddrIsString(addr uint64, size int64) (string, bool) {
	if size <= 0 {
		// wtf?
		return "", false
	}

	if size > int64(e.size) {
		// it's obviously a string cannot larger than file size
		return "", false
	}

	data, err := e.raw.ReadAddr(addr, uint64(size))
	if err != nil {
		return "", false
	}
	if !utf8.Valid(data) {
		return "", false
	}
	return string(data), true
}
