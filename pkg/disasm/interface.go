package disasm

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"github.com/goretk/gore"
	"strconv"
	"unicode/utf8"
)

type PossibleStr struct {
	Addr uint64
	Size uint64
}

type extractorFunc func(code []byte, pc uint64) []PossibleStr

type Extractor struct {
	raw       rawFileWrapper
	size      uint64
	text      []byte        // bytes of text segment (actual instructions)
	textStart uint64        // start PC of text
	textEnd   uint64        // end PC of text
	goarch    string        // GOARCH string
	extractor extractorFunc // disassembler function for goarch
}

type rawFileWrapper interface {
	text() (textStart uint64, text []byte, err error)
	goarch() string
	readAddr(addr, size uint64) ([]byte, error)
}

func NewExtractor(f *gore.GoFile) (*Extractor, error) {
	rawFile := buildWrapper(f)

	size := tool.GetFileSize(f.GetFile())

	textStart, text, err := rawFile.text()
	if err != nil {
		return nil, err
	}

	goarch := rawFile.goarch()
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
		start = e.textStart
	}
	if end > e.textEnd {
		end = e.textEnd
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
		// it's obviously a string can not larger than file size
		return "", false
	}

	data, err := e.raw.readAddr(addr, uint64(size))
	if err != nil {
		return "", false
	}
	if !utf8.Valid(data) {
		return "", false
	}
	return strconv.Quote(string(data)), true
}
