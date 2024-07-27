package disasm

import (
	"debug/dwarf"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type TestFileWrapper struct {
	arch string

	textStart uint64
	text      []byte
	textErr   error
}

func (TestFileWrapper) DWARF() (*dwarf.Data, error) {
	panic("not reachable")
}

func (t TestFileWrapper) Text() (textStart uint64, text []byte, err error) {
	return t.textStart, t.text, t.textErr
}

func (t TestFileWrapper) GoArch() string {
	return t.arch
}

func (TestFileWrapper) ReadAddr(_, _ uint64) ([]byte, error) {
	panic("not reachable")
}

func (TestFileWrapper) LoadSymbols(_ func(name string, addr uint64, size uint64, typ entity.AddrType)) error {
	panic("not reachable")
}

func (TestFileWrapper) LoadSections() *entity.Store {
	panic("not reachable")
}

var _ wrapper.RawFileWrapper = TestFileWrapper{}

func TestNewExtractorNoText(t *testing.T) {
	w := TestFileWrapper{textErr: errors.New("text error")}
	_, err := NewExtractor(w, 0)
	assert.Error(t, err)
}

func TestNewExtractorNoGoArch(t *testing.T) {
	w := TestFileWrapper{}
	_, err := NewExtractor(w, 0)
	assert.ErrorIs(t, err, ErrArchNotSupported)
}

func TestNewExtractorNoExtractor(t *testing.T) {
	w := TestFileWrapper{arch: "unsupported"}
	_, err := NewExtractor(w, 0)
	assert.ErrorIs(t, err, ErrArchNotSupported)
}

func TestExtractor_Extract(t *testing.T) {
	t.Run("start before text", func(t *testing.T) {
		extractor := Extractor{textStart: 0x100, textEnd: 0x200}
		assert.Panics(t, func() {
			extractor.Extract(0x50, 0x100)
		})
	})

	t.Run("end after text", func(t *testing.T) {
		extractor := Extractor{textStart: 0x100, textEnd: 0x200}
		assert.Panics(t, func() {
			extractor.Extract(0x150, 0x250)
		})
	})
}

func TestExtractor_LoadAddrString(t *testing.T) {
	t.Run("size <= 0", func(t *testing.T) {
		extractor := Extractor{}
		ok := extractor.checkAddrString(0, 0)
		assert.False(t, ok)
	})

	t.Run("size > file size", func(t *testing.T) {
		extractor := Extractor{size: 10}
		ok := extractor.checkAddrString(0, 20)
		assert.False(t, ok)
	})
}
