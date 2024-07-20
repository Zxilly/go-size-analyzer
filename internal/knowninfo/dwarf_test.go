package knowninfo

import (
	"debug/dwarf"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeGetEntryValReturnsValueOnSuccess(t *testing.T) {
	entry := &dwarf.Entry{}

	value, ok := safeGetEntryVal[int](entry, dwarf.Attr(1), "test attribute")
	assert.False(t, ok)
	assert.Zero(t, value)
}
