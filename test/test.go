package main

import "reflect"

const ConstInt = 12345
const ConstString = "this is a global const string"
const ConstBool = true
const ConstFloat = 12345.6789
const ConstComplex = 12345.6789 + 12345.6789i
const ConstRune = 'a'

var GlobalInt = 12345
var GlobalString = "12345"
var GlobalBool = true
var GlobalFloat = 12345.6789
var GlobalComplex = 12345.6789 + 12345.6789i
var GlobalRune = 'a'

type ComplexStruct struct {
	Str string
	Num int
}

type ComplexPointerStruct struct {
	Str *string
	Num *int
}

//go:noinline
func UsingComplexStruct() {
	var cs = ComplexStruct{
		Str: "ComplexStruct",
		Num: 12345,
	}
	println(cs.Str)
	println(cs.Num)
}

//go:noinline
func UsingComplexPointerStruct() {
	var cs = ComplexPointerStruct{
		Str: &GlobalString,
		Num: &GlobalInt,
	}
	println(*cs.Str)
	println(*cs.Num)
}

//go:noinline
func UsingConstInt() {
	println(ConstInt)
}

//go:noinline
func UsingGlobalInt() {
	println(GlobalInt)
}

//go:noinline
func UsingConstString() {
	println(ConstString)
}

//go:noinline
func UsingGlobalString() {
	println(GlobalString)
}

//go:noinline
func UsingConstBool() {
	println(ConstBool)
}

//go:noinline
func UsingGlobalBool() {
	//goland:noinspection GoBoolExpressions
	println(GlobalBool)
}

//go:noinline
func UsingConstFloat() {
	println(ConstFloat)
}

//go:noinline
func UsingGlobalFloat() {
	println(GlobalFloat)
}

//go:noinline
func UsingConstComplex() {
	println(ConstComplex)
}

//go:noinline
func UsingGlobalComplex() {
	println(GlobalComplex)
}

//go:noinline
func UsingConstRune() {
	println(ConstRune)
}

//go:noinline
func UsingGlobalRune() {
	println(GlobalRune)
}

//go:noinline
func UsingInLineConstString() {
	const ConstString = "this is a inline const string"
	println(ConstString)
}

//go:noinline
func UsingStackString() {
	var stackString = "this is a stack string"
	println(stackString)
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

	UsingGlobalInt()
	UsingGlobalString()
	UsingGlobalBool()
	UsingGlobalFloat()
	UsingGlobalComplex()
	UsingGlobalRune()
	UsingStackString()

	UsingComplexStruct()
	UsingComplexPointerStruct()

	a := TestStruct{A: 1, B: "2", C: true}
	println(ReflectGetA(a))
}
