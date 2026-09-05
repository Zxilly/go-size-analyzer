package knowninfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// pclnSpans follows cmd/link/internal/ld/pcln.go and runtime._func. Offsets
// remain relative to the original pclntab so shared PC programs stay shared.
type pclnSpans struct {
	stackMap                                           func(uint64, uint64) (entity.AddrPos, bool)
	data                                               []byte
	order                                              binary.ByteOrder
	base, ptr, field, header, names, pcs, funcs, count uint64
	version                                            int
	pcSizes                                            map[uint32]uint64
}

func newPclnSpans(data []byte, base uint64) (*pclnSpans, error) {
	if len(data) < 16 {
		return nil, errors.New("short pclntab header")
	}
	p := &pclnSpans{data: data, base: base, ptr: uint64(data[7]), order: binary.LittleEndian, pcSizes: map[uint32]uint64{}}
	version := func(m uint32) int {
		switch m {
		case 0xfffffffb:
			return 12
		case 0xfffffffa:
			return 116
		case 0xfffffff0:
			return 118
		case 0xfffffff1:
			return 120
		default:
		}
		return 0
	}
	p.version = version(p.order.Uint32(data))
	if p.version == 0 {
		p.order = binary.BigEndian
		p.version = version(p.order.Uint32(data))
	}
	if p.version == 0 || (p.ptr != 4 && p.ptr != 8) {
		return nil, errors.New("unsupported pclntab header")
	}
	p.field = p.ptr
	p.header = 8 + p.ptr
	if p.version >= 118 {
		p.field = 4
		p.header = 8 + 8*p.ptr
	} else if p.version >= 116 {
		p.header = 8 + 7*p.ptr
	}
	if _, err := p.slice(0, p.header); err != nil {
		return nil, err
	}
	word := func(i uint64) uint64 {
		off := 8 + i*p.ptr
		if p.ptr == 4 {
			return uint64(p.order.Uint32(data[off:]))
		}
		return p.order.Uint64(data[off:])
	}
	p.count = word(0)
	if p.version >= 118 {
		p.names = word(3)
		p.pcs = word(6)
		p.funcs = word(7)
	} else if p.version >= 116 {
		p.names = word(2)
		p.pcs = word(5)
		p.funcs = word(6)
	}
	for _, offset := range []uint64{p.names, p.pcs, p.funcs} {
		if _, err := p.slice(offset, 0); err != nil {
			return nil, err
		}
	}
	ftab := p.funcs
	if p.version == 12 {
		ftab = p.header
	}
	if p.count > uint64(len(data))/p.field/2 {
		return nil, errors.New("pclntab function count exceeds file")
	}
	if _, err := p.slice(ftab, (p.count*2+1)*p.field); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *pclnSpans) slice(off, size uint64) ([]byte, error) {
	if off > uint64(len(p.data)) || size > uint64(len(p.data))-off {
		return nil, fmt.Errorf("pclntab range out of bounds: %d+%d", off, size)
	}
	return p.data[off : off+size], nil
}

func (p *pclnSpans) span(off, size uint64) (entity.AddrPos, error) {
	if _, err := p.slice(off, size); err != nil {
		return entity.AddrPos{}, err
	}
	if p.base > ^uint64(0)-off-size {
		return entity.AddrPos{}, errors.New("pclntab address overflow")
	}
	return entity.AddrPos{Addr: p.base + off, Size: size, Type: entity.AddrTypeData}, nil
}

func (p *pclnSpans) pcSize(off uint32) (uint64, error) {
	if off == 0 {
		return 0, nil
	}
	if size, ok := p.pcSizes[off]; ok {
		return size, nil
	}
	start := p.pcs + uint64(off)
	_, err := p.slice(start, 0)
	if err != nil {
		return 0, err
	}
	position := start
	first := true
	for position < uint64(len(p.data)) {
		value, n := binary.Uvarint(p.data[position:])
		if n <= 0 {
			return 0, errors.New("invalid PC value varint")
		}
		position += uint64(n)
		if value == 0 && !first {
			size := position - start
			p.pcSizes[off] = size
			return size, nil
		}
		_, n = binary.Uvarint(p.data[position:])
		if n <= 0 {
			return 0, errors.New("invalid PC delta varint")
		}
		position += uint64(n)
		first = false
	}
	return 0, errors.New("unterminated PC table")
}

func (p *pclnSpans) function(i uint64) ([]entity.AddrPos, entity.PclnSymbolSize, error) {
	var spans []entity.AddrPos
	sizes := entity.NewEmptyPclnSymbolSize()
	if i >= p.count {
		return nil, sizes, errors.New("function index outside pclntab")
	}
	add := func(off, size uint64) error {
		if size == 0 {
			return nil
		}
		r, err := p.span(off, size)
		if err == nil {
			spans = append(spans, r)
		}
		return err
	}
	ftab := p.funcs
	if p.version == 12 {
		ftab = p.header
	}
	entry := ftab + i*2*p.field
	if err := add(entry, 2*p.field); err != nil {
		return nil, sizes, err
	}
	sizes.Ftab = 2 * p.field
	var off uint64
	if p.field == 4 {
		off = uint64(p.order.Uint32(p.data[entry+p.field:]))
	} else {
		off = p.order.Uint64(p.data[entry+p.field:])
	}
	if off > uint64(len(p.data))-p.funcs {
		return nil, sizes, errors.New("function metadata offset outside pclntab")
	}
	off += p.funcs
	lastField := uint64(8)
	if p.version >= 120 {
		lastField = 10
	} else if p.version >= 116 {
		lastField = 9
	}
	headSize := p.field + lastField*4
	h, err := p.slice(off, headSize)
	if err != nil {
		return nil, sizes, err
	}
	field := func(n uint64) uint32 { return p.order.Uint32(h[p.field+(n-1)*4:]) }
	npc, nfd := uint64(field(7)), uint64(h[len(h)-1])
	dataWidth := p.ptr
	if p.version >= 118 {
		dataWidth = 4
	}
	arrayEnd := headSize + npc*4
	if p.version < 118 {
		arrayEnd = (arrayEnd + p.ptr - 1) &^ (p.ptr - 1)
	}
	if err = add(off, arrayEnd+nfd*dataWidth); err != nil {
		return nil, sizes, err
	}
	sizes.Header = headSize
	sizes.FuncData = nfd * dataWidth
	name := p.names + uint64(field(1))
	if _, err = p.slice(name, 0); err != nil {
		return nil, sizes, err
	}
	n := bytes.IndexByte(p.data[name:], 0)
	if n < 0 {
		return nil, sizes, errors.New("unterminated function name")
	}
	sizes.Name = uint64(n + 1)
	if err = add(name, sizes.Name); err != nil {
		return nil, sizes, err
	}
	pc := func(offset uint32) (uint64, error) {
		size, err := p.pcSize(offset)
		if err != nil {
			return 0, err
		}
		return size, add(p.pcs+uint64(offset), size)
	}
	if sizes.PCSP, err = pc(field(4)); err != nil {
		return nil, sizes, err
	}
	if sizes.PCFile, err = pc(field(5)); err != nil {
		return nil, sizes, err
	}
	if sizes.PCLN, err = pc(field(6)); err != nil {
		return nil, sizes, err
	}
	for index := uint64(0); index < npc; index++ {
		offset := p.order.Uint32(p.data[off+headSize+index*4:])
		size, err := pc(offset)
		if err != nil {
			return nil, sizes, err
		}
		sizes.PCData[fmt.Sprintf("pcdata-%d", index)] = int(size + 4)
	}
	if p.stackMap != nil {
		for index := uint64(0); index < min(nfd, 2); index++ {
			cell := off + arrayEnd + index*dataWidth
			var value uint64
			if dataWidth == 4 {
				value = uint64(p.order.Uint32(p.data[cell:]))
			} else {
				value = p.order.Uint64(p.data[cell:])
			}
			if r, ok := p.stackMap(value, p.base+cell); ok {
				spans = append(spans, r)
				sizes.GCMaps += r.Size
			}
		}
	}
	return spans, sizes, nil
}

func (k *KnownInfo) loadExactPclnRanges() error {
	table, err := k.Gore.PCLNTab()
	if err != nil {
		return err
	}
	if len(table.Funcs) == 0 {
		return nil
	}
	p, err := newPclnSpans(table.Funcs[0].LineTable.Data, k.PClnTabAddr)
	if err != nil {
		return err
	}
	if p.count != uint64(len(table.Funcs)) {
		return errors.New("pclntab function count mismatch")
	}
	md, err := k.Gore.Moduledata()
	if err != nil {
		return err
	}
	mapSizes := map[uint64]uint64{}
	p.stackMap = func(value, cell uint64) (entity.AddrPos, bool) {
		addr := value
		if p.version >= 118 {
			base := md.GoFuncValue()
			if value == 0xffffffff || base == 0 || value > ^uint64(0)-base {
				return entity.AddrPos{}, false
			}
			addr = base + value
		} else {
			addr = md.ResolvePointer(value, cell)
		}
		if addr == 0 {
			return entity.AddrPos{}, false
		}
		if size, ok := mapSizes[addr]; ok {
			return entity.AddrPos{Addr: addr, Size: size, Type: entity.AddrTypeData}, size > 0
		}
		mapSizes[addr] = 0
		h, err := k.Wrapper.ReadAddr(addr, 8)
		if err != nil || len(h) != 8 {
			return entity.AddrPos{}, false
		}
		n, bits := int32(p.order.Uint32(h)), int32(p.order.Uint32(h[4:]))
		if n < 0 || bits < 0 {
			return entity.AddrPos{}, false
		}
		size := 8 + uint64(n)*((uint64(bits)+7)/8)
		if size > k.Size || addr+size < addr || !k.Sects.IsData(addr, size) {
			return entity.AddrPos{}, false
		}
		mapSizes[addr] = size
		return entity.AddrPos{Addr: addr, Size: size, Type: entity.AddrTypeData}, true
	}
	functions := make(map[uint64][]*entity.Function)
	for fn := range k.Deps.Functions {
		functions[fn.Addr] = append(functions[fn.Addr], fn)
	}
	for i, raw := range table.Funcs {
		owners := functions[raw.Entry]
		if len(owners) == 0 {
			continue
		}
		ranges, sizes, err := p.function(uint64(i))
		if err != nil {
			return fmt.Errorf("function %s metadata: %w", raw.Name, err)
		}
		for _, fn := range owners {
			fn.PclnSize = sizes
			fn.SetPclnRanges(append(fn.PclnRanges, ranges...))
		}
	}
	pkg := k.getOrCreateVirtualPackage("runtime/pclntab", entity.PackageTypeGenerated)
	k.attributeGeneratedData(pkg, "pclntab:header", p.base, p.header)
	ftab := p.funcs
	if p.version == 12 {
		ftab = p.header
	}
	k.attributeGeneratedData(pkg, "pclntab:ftab-sentinel", p.base+ftab+p.count*2*p.field, p.field)
	return nil
}
