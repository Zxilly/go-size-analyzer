package disasm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type PossibleStr struct {
	Addr     uint64
	Size     uint64
	Header   bool
	TypeAddr uint64
	Copy     bool
}

type extractorFunc func(code []byte, pc uint64) []PossibleStr

type validator func(addr, size uint64) bool

type Extractor struct {
	raw        wrapper.RawFileWrapper
	size       uint64
	text       []byte        // bytes of text segment (actual instructions)
	textStart  uint64        // start PC of text
	textEnd    uint64        // end PC of text
	goarch     string        // GOARCH string
	validators []validator   // validators for possible strings
	extractor  extractorFunc // disassembler function for goarch
	dataCheck  func(addr, size uint64) bool
}

var ErrArchNotSupported = errors.New("unsupported GOARCH")

func NewExtractor(rawFile wrapper.RawFileWrapper,
	size uint64,
	sectCheck func(addr, size uint64) bool,
	goStringSym *entity.AddrPos,
) (*Extractor, error) {
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

	extractor := NewDataExtractor(rawFile, size, sectCheck, goStringSym)
	extractor.text = text
	extractor.textStart = textStart
	extractor.textEnd = textStart + uint64(len(text))
	extractor.goarch = goarch
	extractor.extractor = extractFunc
	return extractor, nil
}

func NewDataExtractor(raw wrapper.RawFileWrapper, size uint64, sectCheck func(uint64, uint64) bool, goStringSym *entity.AddrPos) *Extractor {
	extractor := &Extractor{raw: raw, size: size, dataCheck: sectCheck}
	var validators []validator
	if goStringSym != nil {
		validators = append(validators, func(addr, size uint64) bool {
			return goStringSym.Addr <= addr && addr+size <= goStringSym.Addr+goStringSym.Size
		})
	} else {
		if sectCheck != nil {
			validators = append(validators, sectCheck)
		}
		validators = append(validators, extractor.checkAddrString)
	}

	extractor.validators = validators
	return extractor
}

func (e *Extractor) Validate(addr, size uint64) bool {
	if size == 0 || size > e.size || addr+size < addr {
		return false
	}
	for _, v := range e.validators {
		if !v(addr, size) {
			return false
		}
	}
	return true
}

func (e *Extractor) ValidateReference(p PossibleStr) bool {
	if !p.Copy {
		return e.Validate(p.Addr, p.Size)
	}
	return p.Size > 0 && p.Size <= e.size && p.Addr+p.Size >= p.Addr && e.dataCheck != nil && e.dataCheck(p.Addr, p.Size)
}

func (e *Extractor) Extract(start, end uint64) []PossibleStr {
	if start < e.textStart || end > e.textEnd || start > end {
		slog.Debug("skipping function outside text section", "start", fmt.Sprintf("%#x", start), "end", fmt.Sprintf("%#x", end), "textStart", fmt.Sprintf("%#x", e.textStart), "textEnd", fmt.Sprintf("%#x", e.textEnd))
		return nil
	}

	code := e.text[start-e.textStart : end-e.textStart]

	return e.Resolve(e.extractor(code, start))
}

func (e *Extractor) Resolve(candidates []PossibleStr) []PossibleStr {
	out := candidates[:0]
	for _, candidate := range candidates {
		if candidate.Header {
			if e.dataCheck != nil && !e.dataCheck(candidate.Addr, 16) {
				continue
			}
			if candidate.TypeAddr != 0 {
				// internal/abi.Type: Size_, PtrBytes and Kind_ for a string.
				if e.dataCheck != nil && !e.dataCheck(candidate.TypeAddr, 24) {
					continue
				}
				typ, err := e.raw.ReadAddr(candidate.TypeAddr, 24)
				if err != nil || len(typ) != 24 || binary.LittleEndian.Uint64(typ) != 16 || binary.LittleEndian.Uint64(typ[8:]) != 8 || typ[23]&31 != 24 {
					continue
				}
			}
			header, err := e.raw.ReadAddr(candidate.Addr, 16)
			if err != nil || len(header) != 16 {
				continue
			}
			addr := binary.LittleEndian.Uint64(header)
			if macho, ok := e.raw.(*wrapper.MachoWrapper); ok {
				addr = macho.SlidePointer(addr)
			}
			candidate = PossibleStr{Addr: addr, Size: binary.LittleEndian.Uint64(header[8:])}
		}
		out = append(out, candidate)
	}
	return out
}

func (e *Extractor) checkAddrString(addr, size uint64) bool {
	if size <= 0 {
		// wtf?
		return false
	}

	if size > e.size {
		// it's obviously a string cannot larger than file size
		return false
	}

	data, err := e.raw.ReadAddr(addr, size)
	if err != nil {
		return false
	}
	return utf8.Valid(data)
}
