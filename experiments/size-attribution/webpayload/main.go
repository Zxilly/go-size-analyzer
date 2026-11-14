//go:build !js && !wasm

// Command webpayload projects an existing full JSON report through the current
// web serializer, so comparisons do not conflate analyzer and transport changes.
package main

import (
	"encoding/json/v2"
	"flag"
	"fmt"
	"os"

	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func main() {
	input := flag.String("input", "", "existing full JSON report")
	base := flag.String("output-base", "", "output path prefix")
	flag.Parse()
	if *input == "" || *base == "" {
		fmt.Fprintln(os.Stderr, "-input and -output-base are required")
		os.Exit(1)
	}
	if err := project(*input, *base); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func project(inputPath, base string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	var r result.Result
	err = json.UnmarshalRead(input, &r)
	closeErr := input.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, format := range []string{"html", "web.json", "full-roundtrip.json", "compact.json"} {
		output, err := os.Create(base + ".new." + format)
		if err != nil {
			return err
		}
		switch format {
		case "html":
			err = printer.HTML(&r, output)
		case "web.json":
			err = printer.JSON(&r, output, &printer.JSONOption{FileDetail: true, EscapeForHTML: true})
		case "compact.json":
			err = printer.JSON(&r, output, &printer.JSONOption{HideDetail: true})
		default:
			err = printer.JSON(&r, output, &printer.JSONOption{})
		}
		closeErr := output.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
