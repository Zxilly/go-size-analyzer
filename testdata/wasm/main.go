// Should set GOOS and GOARCH to wasm
//
//go:generate go build -trimpath -o test.wasm main.go
package main

import (
	"fmt"
)

//go:noinline
func helloworld() string {
	return fmt.Sprintf("Hello, world!")
}

func main() {
	println(helloworld())
}
