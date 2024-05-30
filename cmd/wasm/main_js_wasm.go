//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"syscall/js"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer/wasm"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func analyze(_ js.Value, args []js.Value) any {
	utils.InitLogger(slog.LevelDebug)

	name := args[0].String()
	length := args[1].Length()
	data := make([]byte, length)
	js.CopyBytesToGo(data, args[1])

	reader := bytes.NewReader(data)

	result, err := internal.Analyze(name, reader, uint64(length), internal.Options{
		SkipDisasm: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return js.ValueOf(nil)
	}

	return wasm.JavaScript(result)
}

func main() {
	utils.ApplyMemoryLimit()

	js.Global().Set("gsa_analyze", js.FuncOf(analyze))
	js.Global().Get("console").Call("log", "Go size analyzer initialized")

	select {}
}
