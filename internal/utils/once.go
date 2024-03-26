package utils

import "sync"

type ValueOnce[T any] struct {
	*sync.Once
	v T
}

func NewOnce[T any]() *ValueOnce[T] {
	return &ValueOnce[T]{Once: &sync.Once{}}
}

func (o *ValueOnce[T]) Set(v T) {
	o.v = v
}

func (o *ValueOnce[T]) Get() T {
	return o.v
}
