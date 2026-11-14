package dwarf

import (
	"debug/dwarf"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// Traverse statically typed aggregates and initialized, file-backed pointers.
// These are references from a variable, not declarations of the target objects.
// Interfaces and other runtime-managed containers need a dynamic-type model.
type typedDataWalker struct {
	memory     MemoryReader
	references map[dwarf.Type]bool
	contents   []Content
	visited    int
	err        error
	objects    map[typedObject]bool
	targets    map[typedObject]bool
}

type typedObject struct {
	addr uint64
	typ  dwarf.Type
}

// Go's DW_AT_go_kind distinguishes slices/strings and runtime-managed types
// from user structs. debug/dwarf does not retain this extension on Type.
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
		if !ok || (kind < 18 || kind > 24) && kind != 26 {
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
	if t == nil || depth >= 32 || w.runtimeManaged(t) {
		return false
	}
	if value, ok := w.references[t]; ok {
		return value
	}
	w.references[t] = false
	result := false
	switch typ := underlying(t).(type) {
	case *dwarf.PtrType:
		result = w.pointerTarget(typ) != nil
	case *dwarf.StructType:
		if typ.Incomplete || typ.Kind == "union" {
			return false
		}
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

func (w *typedDataWalker) runtimeManaged(t dwarf.Type) bool {
	for _, kind := range []int64{w.memory.Kinds[t], w.memory.Kinds[underlying(t)]} {
		switch kind {
		case 18, 19, 20, 21, 26: // chan, func, interface, map, unsafe.Pointer
			return true
		}
	}
	if typ, ok := underlying(t).(*dwarf.StructType); ok {
		return typ.StructName == "runtime.eface" || typ.StructName == "runtime.iface"
	}
	return false
}

func (w *typedDataWalker) pointerTarget(t *dwarf.PtrType) dwarf.Type {
	if t == nil || t.Type == nil || (t.Size() != 4 && t.Size() != 8) || w.runtimeManaged(t.Type) {
		return nil
	}
	target := underlying(t.Type)
	switch target.(type) {
	case nil, *dwarf.VoidType, *dwarf.FuncType, *dwarf.UnspecifiedType:
		return nil
	}
	if aggregate, ok := target.(*dwarf.StructType); ok && (aggregate.Incomplete || aggregate.Kind == "union") {
		return nil
	}
	if size, ok := checkedTypeSize(target, 0); !ok || size <= 0 {
		return nil
	}
	return target
}

// ArrayType.Size recursively multiplies signed lengths without overflow or
// cycle checks. Validate newly followed targets before asking it for a size.
func checkedTypeSize(t dwarf.Type, depth int) (int64, bool) {
	if depth >= 32 || t == nil {
		return 0, false
	}
	t = underlying(t)
	if t == nil {
		return 0, false
	}
	if array, ok := t.(*dwarf.ArrayType); ok {
		if array.Count < 0 {
			return 0, false
		}
		size, valid := checkedTypeSize(array.Type, depth+1)
		if !valid || size < 0 || size != 0 && array.Count > math.MaxInt64/size {
			return 0, false
		}
		return array.Count * size, true
	}
	return t.Size(), t.Size() >= 0
}

func (w *typedDataWalker) pointer(t *dwarf.PtrType, addr uint64, prefix string, depth int) {
	target := w.pointerTarget(t)
	if target == nil {
		return
	}
	data, err := w.memory.readExact(addr, uint64(t.Size()))
	if err != nil {
		return
	} // BSS and unavailable static pointer fields add no evidence.
	pointer, err := readUintTo64(data, w.memory.ByteOrder)
	if err != nil || pointer == 0 {
		return
	}
	resolved := w.memory.resolve(pointer)
	key := typedObject{addr: resolved, typ: target}
	if w.objects[key] || w.targets[key] {
		return
	}
	checkpoint := len(w.contents)
	if !w.add(prefix, "*", pointer, uint64(target.Size())) {
		return
	}
	if w.targets == nil {
		w.targets = make(map[typedObject]bool)
	}
	w.targets[key] = true
	w.walk(target, resolved, joinContent(prefix, "*"), depth+1)
	if w.err != nil {
		// A newly followed pointer must not discard a variable's other valid
		// fields when its target contains an invalid or unsupported header.
		w.contents = w.contents[:checkpoint]
		w.err = nil
	}
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
	key := typedObject{addr: addr, typ: underlying(t)}
	if w.objects == nil {
		w.objects = make(map[typedObject]bool)
	}
	if w.objects[key] {
		return
	}
	w.objects[key] = true
	w.visited++
	switch typ := underlying(t).(type) {
	case *dwarf.PtrType:
		w.pointer(typ, addr, prefix, depth)
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
