// A known two-parent retention graph. Do not inline the three functions.
package main

var sink int

//go:noinline
func shared(x int) int { return x + sink }

//go:noinline
func callerA(x int) int { return shared(x) + 1 }

//go:noinline
func callerB(x int) int { return shared(x) + 2 }

func main() { sink = callerA(11) + callerB(22) }
