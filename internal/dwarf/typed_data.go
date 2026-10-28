package dwarf

import (
	"debug/dwarf"
	"errors"
	"fmt"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// Traverse only statically typed aggregates. Ordinary pointers and interfaces
// are not chased: they require a separate ownership and dynamic-type model.
type typedDataWalker struct {
	memory     MemoryReader
	references map[dwarf.Type]bool
	contents   []Content
	visited    int
	err        error
}

// Go's DW_AT_go_kind distinguishes named slices/strings from user structs
// with identical fields. debug/dwarf does not retain this extension on Type.
func ReadGoTypeKinds(d *dwarf.Data) (map[dwarf.Type]int64, error) {
	return scanGoTypeKinds(d, nil)
}

func scanGoTypeKinds(d *dwarf.Data, visit func(*dwarf.Entry)) (map[dwarf.Type]int64, error) {
	kinds := map[dwarf.Type]int64{}
	r := d.Reader()
	for {
		e, err := r.Next()
		if err != nil {
			return nil, err
		}
		if e == nil {
			break
		}
		if visit != nil {
			visit(e)
		}
		kind, ok := e.Val(dwarf.Attr(0x2900)).(int64)
		if !ok || (kind != 23 && kind != 24) {
			continue
		}
		t, err := d.Type(e.Offset)
		if err == nil {
			kinds[t] = kind
			if base := underlying(t); base != nil {
				kinds[base] = kind
			}
		}
	}
	return kinds, nil
}

func (w *typedDataWalker) isString(t *dwarf.StructType) bool {
	return t.StructName == "string" || w.memory.Kinds[t] == 24
}

func (w *typedDataWalker) isSlice(t *dwarf.StructType) bool {
	return strings.HasPrefix(t.StructName, "[]") || w.memory.Kinds[t] == 23
}

func underlying(t dwarf.Type) dwarf.Type {
	for range 32 {
		switch v := t.(type) {
		case *dwarf.TypedefType:
			t = v.Type
		case *dwarf.QualType:
			t = v.Type
		default:
			return t
		}
	}
	return nil
}

func (w *typedDataWalker) hasReferences(t dwarf.Type, depth int) bool {
	if t == nil || depth >= 32 {
		return false
	}
	if value, ok := w.references[t]; ok {
		return value
	}
	w.references[t] = false
	result := false
	switch typ := underlying(t).(type) {
	case *dwarf.StructType:
		result = w.isString(typ) || typ.StructName == "embed.FS" || w.isSlice(typ)
		for i, field := range typ.Field {
			if field == nil || field.Type == nil {
				w.err = fmt.Errorf("field %d has no type", i)
				return false
			}
			result = result || w.hasReferences(field.Type, depth+1)
		}
	case *dwarf.ArrayType:
		result = w.hasReferences(typ.Type, depth+1)
	default:
	}
	w.references[t] = result
	return result
}

func joinContent(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func (w *typedDataWalker) add(prefix, name string, addr, size uint64) bool {
	if size == 0 || addr == 0 {
		return false
	}
	resolved := w.memory.resolve(addr)
	if size > ^uint64(0)-resolved || (w.memory.Limit != 0 && size > w.memory.Limit) {
		return false
	}
	if w.memory.Valid != nil {
		if !w.memory.Valid(resolved, size) {
			return false
		}
	} else if _, err := w.memory.readExact(resolved, size); err != nil {
		return false
	}
	w.contents = append(w.contents, Content{Name: joinContent(prefix, name), Addr: addr, Size: size})
	return true
}

func (w *typedDataWalker) walk(t dwarf.Type, addr uint64, prefix string, depth int) {
	if w.err != nil || depth >= 32 || w.visited >= 8192 {
		return
	}
	if !w.hasReferences(t, 0) || w.err != nil {
		return
	}
	w.visited++
	switch typ := underlying(t).(type) {
	case *dwarf.StructType:
		switch {
		case w.isString(typ):
			pointer, size, err := readString(typ, addr, w.memory)
			if err != nil {
				w.err = err
				return
			}
			w.add(prefix, "string", pointer, size)
		case typ.StructName == "embed.FS":
			parts, err := readEmbedFS(typ, addr, w.memory)
			if err != nil {
				w.err = err
				return
			}
			for _, part := range parts {
				w.add(prefix, part.Name, part.Addr, part.Size)
			}
		case w.isSlice(typ):
			w.slice(typ, addr, prefix, depth)
		default:
			for _, field := range typ.Field {
				if field == nil || field.ByteOffset < 0 || field.BitSize != 0 || field.Type == nil || field.Type.Size() < 0 || field.ByteOffset > typ.Size() || field.Type.Size() > typ.Size()-field.ByteOffset || uint64(field.ByteOffset) > ^uint64(0)-addr {
					continue
				}
				w.walk(field.Type, addr+uint64(field.ByteOffset), joinContent(prefix, field.Name), depth+1)
			}
		}
	case *dwarf.ArrayType:
		if typ.Count <= 0 || typ.Type == nil || typ.Type.Size() <= 0 || (typ.StrideBitSize != 0 && typ.StrideBitSize != typ.Type.Size()*8) {
			return
		}
		stride := uint64(typ.Type.Size())
		if uint64(typ.Count) > ^uint64(0)/stride || uint64(typ.Count)*stride > ^uint64(0)-addr {
			return
		}
		for i := int64(0); i < typ.Count && w.visited < 8192 && w.err == nil; i++ {
			w.walk(typ.Type, addr+uint64(i)*stride, fmt.Sprintf("%s[%d]", prefix, i), depth+1)
		}
	default:
	}
}

func (w *typedDataWalker) slice(t *dwarf.StructType, addr uint64, prefix string, depth int) {
	if t == nil || len(t.Field) != 3 || t.Field[0] == nil || t.Field[0].Type == nil {
		w.err = errors.New("slice header has no array field")
		return
	}
	if err := checkHeader(t, fieldPattern{"array", t.Field[0].Type.String()}, fieldPattern{"len", "int"}, fieldPattern{"cap", "int"}); err != nil {
		w.err = err
		return
	}
	array, ok := underlying(t.Field[0].Type).(*dwarf.PtrType)
	if !ok || array.Type == nil || array.Type.Size() < 0 {
		w.err = errors.New("slice array field has no valid element type")
		return
	}
	if array.Type.Size() == 0 {
		return
	}
	values, err := readHeader(t, addr, w.memory)
	if errors.Is(err, wrapper.ErrAddrNotFound) {
		return
	}
	if err != nil {
		w.err = err
		return
	}
	length, err := signedLength(values[1], t.Field[1].Type.Size())
	if err != nil {
		w.err = err
		return
	}
	capacity, err := signedLength(values[2], t.Field[2].Type.Size())
	if err != nil {
		w.err = err
		return
	}
	stride := uint64(array.Type.Size())
	if capacity < length || capacity > ^uint64(0)/stride {
		w.err = errors.New("invalid slice length or capacity")
		return
	}
	pointer := values[0]
	if !w.add(prefix, "[]"+array.Type.String(), pointer, capacity*stride) || !w.hasReferences(array.Type, 0) {
		return
	}
	base := w.memory.resolve(pointer)
	for i := uint64(0); i < length && w.visited < 8192 && w.err == nil; i++ {
		w.walk(array.Type, base+i*stride, fmt.Sprintf("%s[%d]", prefix, i), depth+1)
	}
}
