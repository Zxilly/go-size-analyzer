package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/arch/x86/x86asm"
)

type fact struct {
	value uint64
	kind  uint8
	cross bool
}
type (
	state       [16]fact
	instruction struct {
		pc     uint64
		inst   x86asm.Inst
		next   uint64
		succ   []uint64
		branch bool
	}
)

func register(r x86asm.Reg) int {
	if r >= x86asm.RAX && r <= x86asm.R15 {
		return int(r - x86asm.RAX)
	}
	if r >= x86asm.EAX && r <= x86asm.R15L {
		return int(r - x86asm.EAX)
	}
	return -1
}

func operand(a x86asm.Arg, s state) fact {
	switch v := a.(type) {
	case x86asm.Reg:
		if i := register(v); i >= 0 {
			return s[i]
		}
	case x86asm.Imm:
		return fact{value: uint64(v), kind: 1}
	default:
		return fact{}
	}
	return fact{}
}

func transfer(in instruction, s state) state {
	op := in.inst.Op.String()
	dst, reg := in.inst.Args[0].(x86asm.Reg)
	i := register(dst)
	switch op {
	case "MOV":
		if reg && i < 0 {
			return state{}
		}
		if reg && i >= 0 {
			s[i] = operand(in.inst.Args[1], s)
			if dst >= x86asm.EAX && dst <= x86asm.R15L {
				s[i].value = uint64(uint32(s[i].value))
			}
		} // Memory writes do not change tracked register values.
	case "LEA":
		if reg && i < 0 {
			return state{}
		}
		if reg && i >= 0 {
			v := fact{}
			if mem, ok := in.inst.Args[1].(x86asm.Mem); ok && mem.Index == 0 && mem.Segment == 0 {
				if mem.Base == x86asm.RIP {
					v = fact{value: uint64(int64(in.next) + mem.Disp), kind: 2}
				} else if j := register(mem.Base); j >= 0 && s[j].kind == 2 {
					v = s[j]
					v.value = uint64(int64(v.value) + mem.Disp)
				}
			}
			s[i] = v
			if dst >= x86asm.EAX && dst <= x86asm.R15L {
				s[i].value = uint64(uint32(s[i].value))
			}
		}
	case "XOR":
		if reg && i < 0 {
			return state{}
		}
		if reg && i >= 0 {
			if src, ok := in.inst.Args[1].(x86asm.Reg); ok && register(src) == i {
				s[i] = fact{kind: 1}
			} else {
				s[i] = fact{}
			}
		}
	case "CMP", "TEST", "NOP", "ENDBR64", "JMP", "RET":
	default:
		if strings.HasPrefix(op, "J") {
			return s
		}
		return state{}
	}
	return s
}

func meet(a, b state) state {
	var out state
	for i := range a {
		if a[i].kind != 0 && a[i].kind == b[i].kind && a[i].value == b[i].value {
			out[i] = a[i]
			out[i].cross = a[i].cross || b[i].cross
		}
	}
	return out
}

func scanCFG(code []byte, base uint64, root string, m imageFile) ([]candidate, bool) {
	if len(code) > 128<<10 {
		return nil, true
	}
	nodes := map[uint64]instruction{}
	pcs := []uint64{}
	for offset := 0; offset < len(code); {
		inst, err := x86asm.Decode(code[offset:], 64)
		if err != nil || inst.Len == 0 {
			return nil, false
		}
		pc := base + uint64(offset)
		n := instruction{pc: pc, inst: inst, next: pc + uint64(inst.Len)}
		op := inst.Op.String()
		switch {
		case op == "RET" || op == "UD2":
		case op == "JMP":
			if rel, ok := inst.Args[0].(x86asm.Rel); ok {
				n.succ = []uint64{uint64(int64(n.next) + int64(rel))}
				n.branch = true
			}
		case strings.HasPrefix(op, "J"):
			if rel, ok := inst.Args[0].(x86asm.Rel); ok {
				n.succ = []uint64{n.next, uint64(int64(n.next) + int64(rel))}
				n.branch = true
			}
		default:
			n.succ = []uint64{n.next}
		}
		nodes[pc] = n
		pcs = append(pcs, pc)
		offset += inst.Len
	}
	if len(nodes) > 15000 {
		return nil, true
	}
	states := map[uint64]state{base: {}}
	queue := []uint64{base}
	steps := 0
	for len(queue) > 0 {
		pc := queue[0]
		queue = queue[1:]
		n, ok := nodes[pc]
		if !ok {
			continue
		}
		s := states[pc]
		out := transfer(n, s)
		if n.inst.Op == x86asm.CALL {
			out = state{}
		}
		if n.branch {
			for i := range out {
				if out[i].kind != 0 {
					out[i].cross = true
				}
			}
		}
		for _, next := range n.succ {
			if _, ok := nodes[next]; !ok {
				continue
			}
			old, seen := states[next]
			merged := out
			if seen {
				merged = meet(old, out)
			}
			if !seen || merged != old {
				states[next] = merged
				queue = append(queue, next)
			}
		}
		steps++
		if steps > 60000 {
			return nil, true
		}
	}
	abi := []x86asm.Reg{x86asm.RAX, x86asm.RBX, x86asm.RCX, x86asm.RDI, x86asm.RSI, x86asm.R8, x86asm.R9, x86asm.R10, x86asm.R11}
	out := []candidate{}
	seen := map[string]bool{}
	for _, pc := range pcs {
		n := nodes[pc]
		if n.inst.Op != x86asm.CALL && n.inst.Op != x86asm.RET {
			continue
		}
		s, ok := states[pc]
		if !ok {
			continue
		}
		for i := 1; i < len(abi); i++ {
			p, l := s[register(abi[i-1])], s[register(abi[i])]
			if p.kind != 2 || l.kind != 1 || l.value < 4 || l.value > 1<<20 {
				continue
			}
			data, ok := m.read(p.value, l.value)
			if !ok || !utf8.Valid(data) {
				continue
			}
			printable := true
			for _, r := range string(data) {
				if !unicode.IsPrint(r) && r != '\n' && r != '\r' && r != '\t' {
					printable = false
					break
				}
			}
			if !printable {
				continue
			}
			sp, ok := m.mapAddr(p.value, l.value)
			if !ok {
				continue
			}
			cross := p.cross || l.cross
			key := fmt.Sprintf("%d:%d:%t", p.value, l.value, cross)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, candidate{span: sp, Address: p.value, Root: root, Path: fmt.Sprintf("PC %#x ABI slots %d/%d", pc, i-1, i), Kind: "cfg_printable_string_candidate", CrossBlock: cross})
		}
	}
	return out, false
}
