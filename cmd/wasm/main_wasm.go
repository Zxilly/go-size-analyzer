//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"syscall/js"
	"unsafe"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func analyze(_ js.Value, args []js.Value) any {
	utils.InitLogger(slog.LevelDebug)

	name := args[0].String()
	data := make([]byte, args[1].Length())
	js.CopyBytesToGo(data, args[1])

	reader := bytes.NewReader(data)

	result, err := internal.Analyze(name, reader, uint64(len(data)), internal.Options{
		SkipDisasm: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return js.ValueOf(nil)
	}

	buf := new(bytes.Buffer)
	err = printer.JSON(result, &printer.JSONOption{
		Writer:     buf,
		Indent:     nil,
		HideDetail: true,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return js.ValueOf(nil)
	}

	return js.ValueOf(unsafe.String(unsafe.SliceData(buf.Bytes()), buf.Len()))
}

func main() {
	utils.ApplyMemoryLimit()

	js.Global().Set("gsa_analyze", js.FuncOf(analyze))

	select {}
}
