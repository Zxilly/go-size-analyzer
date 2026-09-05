package knowninfo

import (
	"log/slog"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// AnalyzeStaticHeaders follows initialized pointer/length pairs belonging to
// named variables. It does not scan arbitrary aligned words across the file.
func (k *KnownInfo) AnalyzeStaticHeaders() {
	ptr, order := ptrSizeAndOrder(k.Wrapper.GoArch())
	e := disasm.NewDataExtractor(k.Wrapper, k.Size, k.Sects.IsData, k.GoStringSymbol)
	count := 0
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		var headers []*entity.Addr
		for addr := range p.SymbolAddresses {
			if addr.Symbol == nil || (addr.SourceType != entity.AddrSourceSymbol && addr.SourceType != entity.AddrSourceDwarf) || (addr.Size != uint64(ptr*2) && addr.Size != uint64(ptr*3)) {
				continue
			}
			if strings.HasPrefix(addr.Symbol.Name, "go:") || strings.HasPrefix(addr.Symbol.Name, "type:") {
				continue
			}
			headers = append(headers, addr)
		}
		for _, header := range headers {
			data, err := k.Wrapper.ReadAddr(header.Addr, header.Size)
			if err != nil || uint64(len(data)) != header.Size {
				continue
			}
			word := func(off int) uint64 {
				if ptr == 4 {
					return uint64(order.Uint32(data[off:]))
				}
				return order.Uint64(data[off:])
			}
			addr, size := k.convertAddr(word(0)), word(ptr)
			if header.Size == uint64(ptr*3) && word(ptr*2) != size {
				continue
			}
			if !e.Validate(addr, size) {
				continue
			}
			if old, ok := k.KnownAddr.SymbolAddrSpace.Get(addr); ok && old.Pkg == p && old.Size >= size {
				continue
			}
			sym := entity.NewSymbol(header.Symbol.Name+".data", addr, size, entity.AddrTypeData)
			if ap := k.KnownAddr.InsertSymbol(sym, p, entity.AddrSourceStaticHeader); ap != nil {
				p.AddSymbol(sym, ap)
				count++
			}
		}
		return nil
	})
	slog.Info("Static variable headers analyzed", "references", count)
}
