package main

import (
	"go.szostok.io/version/printer"
	"strings"
)

func GetVersion() string {
	p := printer.New()
	s := new(strings.Builder)
	err := p.Print(s)
	if err != nil {
		panic(err)
	}
	return s.String()
}
