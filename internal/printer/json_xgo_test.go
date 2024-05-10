//go:build xgo

package printer

import (
	"encoding/json"
	"errors"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/xhd2015/xgo/runtime/mock"
	"testing"
)

func TestJson(t *testing.T) {
	mock.Patch(json.Marshal, func(v any) ([]byte, error) {
		return nil, errors.New("mocked")
	})

	mock.Patch(json.MarshalIndent, func(v any, prefix, indent string) ([]byte, error) {
		return nil, errors.New("mocked")
	})

	called := false
	mock.Patch(utils.FatalError, func(err error) {
		called = true
	})

	Json(nil, nil)

	assert.True(t, called)
}
