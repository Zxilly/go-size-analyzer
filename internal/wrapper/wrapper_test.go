package wrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWrapper(t *testing.T) {
	ret := NewWrapper(nil)
	assert.Nil(t, ret)
}
