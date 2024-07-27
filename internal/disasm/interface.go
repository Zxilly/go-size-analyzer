package disasm

import (
	"fmt"
	"unicode/utf8"

	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type PossibleStr struct {
	Addr uint64
	Size uint64
}

type extractorFunc func(code []byte, pc uint64) []PossibleStr

type validator func(addr uint64, size int64) bool

type Extractor struct {
	raw        wrapper.RawFileWrapper
	size       uint64
	text       []byte        // bytes of text segment (actual instructions)
	textStart  uint64        // start PC of text
	textEnd    uint64        // end PC of text
	goarch     string        // GOARCH string
	validators []validator   // validators for possible strings
	extractor  extractorFunc // disassembler function for goarch
}

var ErrArchNotSupported = fmt.Errorf("unsupported GOARCH")

func NewExtractor(rawFile wrapper.RawFileWrapper, size uint64) (*Extractor, error) {
	textStart, text, err := rawFile.Text()
	if err != nil {
		return nil, err
	}

	goarch := rawFile.GoArch()
	if goarch == "" {
		return nil, ErrArchNotSupported
	}
	extractFunc := extractFuncs[goarch]
	if extractFunc == nil {
		return nil, fmt.Errorf("%w %s", ErrArchNotSupported, goarch)
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
		panic(fmt.Errorf("start address %#x is before text segment %#x", start, e.textStart))
	}
	if end > e.textEnd {
		panic(fmt.Errorf("end address %#x is after text segment %#x", end, e.textEnd))
	}

	code := e.text[start-e.textStart : end-e.textStart]

	return e.extractor(code, start)
}

func (e *Extractor) CheckAddrString(addr uint64, size int64) bool {
	if size <= 0 {
		// wtf?
		return false
	}

	if size > int64(e.size) {
		// it's obviously a string cannot larger than file size
		return false
	}

	data, err := e.raw.ReadAddr(addr, uint64(size))
	if err != nil {
		return false
	}
	return utf8.Valid(data)
}
