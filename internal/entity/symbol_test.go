package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/assert"
)

func TestSymbolStringRepresentation(t *testing.T) {
	symbol := &entity.Symbol{
		Name: "testSymbol",
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Symbol: testSymbol Addr: 1000 CodeSize: 100 Type: data"
	result := symbol.String()

	assert.Equal(t, expected, result)
}

func TestSymbolStringRepresentationWithDifferentType(t *testing.T) {
	symbol := &entity.Symbol{
		Name: "testSymbol",
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeText,
	}

	expected := "Symbol: testSymbol Addr: 1000 CodeSize: 100 Type: text"
	result := symbol.String()

	assert.Equal(t, expected, result)
}

func TestSymbolStringRepresentationWithZeroSize(t *testing.T) {
	symbol := &entity.Symbol{
		Name: "testSymbol",
		Addr: 4096,
		Size: 0,
		Type: entity.AddrTypeData,
	}

	expected := "Symbol: testSymbol Addr: 1000 CodeSize: 0 Type: data"
	result := symbol.String()

	assert.Equal(t, expected, result)
}

func TestSymbolStringRepresentationWithZeroAddr(t *testing.T) {
	symbol := &entity.Symbol{
		Name: "testSymbol",
		Addr: 0,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Symbol: testSymbol Addr: 0 CodeSize: 100 Type: data"
	result := symbol.String()

	assert.Equal(t, expected, result)
}

func TestNewSymbolCreation(t *testing.T) {
	name := "testSymbol"
	addr := uint64(4096)
	size := uint64(256)
	typ := entity.AddrTypeData

	symbol := entity.NewSymbol(name, addr, size, typ)

	assert.Equal(t, name, symbol.Name)
	assert.Equal(t, addr, symbol.Addr)
	assert.Equal(t, size, symbol.Size)
	assert.Equal(t, typ, symbol.Type)
}
