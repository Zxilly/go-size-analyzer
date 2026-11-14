package main

import "os"

var Sink int
var ByteSink byte

type Count int

var CountSink Count

func addOne(v int) int {
	for i := 0; i < 4; i++ {
		v = v*1664525 + 1013904223
	}
	return v + 1
}
func doubled(v int) int { return addOne(v) * 2 }

//go:noinline
func host(v int) int { return doubled(v) + addOne(v) }

//go:noinline
func genericFirst[T ~int | ~byte](v []T) T { return v[0] }

func main() {
	Sink = host(len(os.Args))
	Sink += genericFirst([]int{1, 2})
	ByteSink = genericFirst([]byte{3, 4})
	CountSink = genericFirst([]Count{5, 6})
}
