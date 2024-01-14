// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package objfile

import (
	"debug/gosym"
	"encoding/binary"
	"fmt"
	"golang.org/x/arch/x86/x86asm"
)

// Disasm is a disassembler for a given File.
type Disasm struct {
	pcln      Liner            // pcln table
	text      []byte           // bytes of text segment (actual instructions)
	textStart uint64           // start PC of text
	textEnd   uint64           // end PC of text
	goarch    string           // GOARCH string
	check     checkAsmFunc     // disassembler function for goarch
	byteOrder binary.ByteOrder // byte order for goarch
}

// Disasm returns a disassembler for the file f.
func (e *Entry) Disasm() (*Disasm, error) {
	pcln, err := e.PCLineTable()
	if err != nil {
		return nil, err
	}

	textStart, textBytes, err := e.Text()
	if err != nil {
		return nil, err
	}

	goarch := e.GOARCH()
	disasm := checks[goarch]
	byteOrder := byteOrders[goarch]
	if disasm == nil || byteOrder == nil {
		return nil, fmt.Errorf("unsupported architecture")
	}

	d := &Disasm{
		pcln:      pcln,
		text:      textBytes,
		textStart: textStart,
		textEnd:   textStart + uint64(len(textBytes)),
		goarch:    goarch,
		check:     disasm,
		byteOrder: byteOrder,
	}

	return d, nil
}

// Filter the .rodata string address and length
func (d *Disasm) Filter(start, end uint64) {
	if start < d.textStart {
		start = d.textStart
	}
	if end > d.textEnd {
		end = d.textEnd
	}
	code := d.text[:end-d.textStart]

	expectLen := false

	stringFound := make([]struct {
		Start uint64
		Len   uint64
	}, 0)

	var lastAddr uint64 = 0

	for pc := start; pc < end; {
		i := pc - d.textStart
		ret, size := d.check(code[i:], pc, expectLen, d.byteOrder)
		if ret.typ == FoundPossibleAddr {
			expectLen = true
			lastAddr = ret.value
		} else {
			expectLen = false
			if ret.typ == FoundLength {
				stringFound = append(stringFound, struct {
					Start uint64
					Len   uint64
				}{Start: lastAddr, Len: ret.value})
			} else {
				lastAddr = 0
			}
		}

		pc += uint64(size)
	}
}

type judgeType int

const (
	FoundPossibleAddr judgeType = iota
	FoundLength
	NotFound
)

type result struct {
	typ   judgeType
	value uint64 // addr or length
}

type checkAsmFunc func(code []byte, pc uint64, expectLen bool, ord binary.ByteOrder) (result, int)

func check386(code []byte, pc uint64, expectLen bool, _ binary.ByteOrder) (result, int) {
	return checkX86(code, pc, expectLen, 32)
}

var _ checkAsmFunc = checkAmd64

func checkAmd64(code []byte, pc uint64, expectLen bool, _ binary.ByteOrder) (result, int) {
	return checkX86(code, pc, expectLen, 64)
}

func checkX86(code []byte, pc uint64, expectLen bool, arch int) (result, int) {
	inst, err := x86asm.Decode(code, arch)

	size := inst.Len
	if err != nil || size == 0 || inst.Op == 0 {
		return result{NotFound, 0}, 1
	}

	if inst.Op == x86asm.L

	text := x86asm.GoSyntax(inst, pc, nil)
	println(text)

	return result{NotFound, 0}, size
}

//func disasm_arm(code []byte, pc uint64, _ binary.ByteOrder) (result, int) {
//	inst, err := armasm.Filter(code, armasm.ModeARM)
//	var text string
//	size := inst.Len
//	if err != nil || size == 0 || inst.Op == 0 {
//		size = 4
//		text = "?"
//	}
//	return text, size
//}
//
//func disasm_arm64(code []byte, pc uint64, byteOrder binary.ByteOrder) (result, int) {
//	inst, err := arm64asm.Filter(code)
//	var text string
//	if err != nil || inst.Op == 0 {
//		text = "?"
//	}
//	return
//}
//
//func disasm_ppc64(code []byte, pc uint64,  byteOrder binary.ByteOrder) (result, int) {
//	inst, err := ppc64asm.Filter(code, byteOrder)
//	var text string
//	size := inst.Len
//	if err != nil || size == 0 {
//		size = 4
//		text = "?"
//	} else {
//
//	}
//	return text, size
//}

var checks = map[string]checkAsmFunc{
	//"386":     disasm_386,
	"amd64": checkAmd64,
	//"arm":     disasm_arm,
	//"arm64":   disasm_arm64,
	//"ppc64":   disasm_ppc64,
	//"ppc64le": disasm_ppc64,
}

var byteOrders = map[string]binary.ByteOrder{
	"386":     binary.LittleEndian,
	"amd64":   binary.LittleEndian,
	"arm":     binary.LittleEndian,
	"arm64":   binary.LittleEndian,
	"ppc64":   binary.BigEndian,
	"ppc64le": binary.LittleEndian,
	"s390x":   binary.BigEndian,
}

type Liner interface {
	// PCToLine Given a pc, returns the corresponding file, line, and function data.
	// If unknown, returns "",0,nil.
	PCToLine(uint64) (string, int, *gosym.Func)
}
