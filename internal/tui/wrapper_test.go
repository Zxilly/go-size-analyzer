package tui

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func Test_newWrapper(t *testing.T) {
	assert.Panics(t, func() {
		newWrapper(nil)
	})
}

func Test_wrapper_Description(t *testing.T) {
	w := wrapper{}
	assert.Panics(t, func() {
		w.Description()
	})
}

func Test_wrapper_Title(t *testing.T) {
	w := wrapper{}
	assert.Panics(t, func() {
		w.Title()
	})

	w = wrapper{function: &entity.Function{
		Type: "invalid",
	}}

	assert.Panics(t, func() {
		w.Title()
	})
}

func Test_wrapper_children(t *testing.T) {
	w := wrapper{cacheOnce: &sync.Once{}}
	assert.Panics(t, func() {
		w.children()
	})
}

func Test_wrapper_size(t *testing.T) {
	w := wrapper{}
	assert.Panics(t, func() {
		w.size()
	})
}
