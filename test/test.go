package main

import "reflect"

const ConstInt = 12345
const ConstString = "12345"
const ConstBool = true
const ConstFloat = 12345.6789
const ConstComplex = 12345.6789 + 12345.6789i
const ConstRune = 'a'

//go:noinline
func UsingConstInt() {
	println(ConstInt)
}

//go:noinline
func UsingConstString() {
	println(ConstString)
}

//go:noinline
func UsingConstBool() {
	println(ConstBool)
}

//go:noinline
func UsingConstFloat() {
	println(ConstFloat)
}

//go:noinline
func UsingConstComplex() {
	println(ConstComplex)
}

//go:noinline
func UsingConstRune() {
	println(ConstRune)
}

//go:noinline
func UsingInLineConstString() {
	const ConstString = "this is a inline const string"
	println(ConstString)
}

type TestStruct struct {
	A int
	B string
	C bool
}

func ReflectGetA(t TestStruct) int {
	// this disables the dead code elimination of gc
	return int(reflect.ValueOf(t).FieldByName("A").Int())
}

func main() {
	UsingConstInt()
	UsingConstString()
	UsingConstBool()
	UsingConstFloat()
	UsingConstComplex()
	UsingConstRune()
	UsingInLineConstString()

	a := TestStruct{A: 1, B: "2", C: true}
	println(ReflectGetA(a))
}
