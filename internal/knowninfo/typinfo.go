package knowninfo

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// AnalyzeTypes extracts type descriptors from the binary's moduledata and
// attributes them as data symbols to the appropriate packages.
// It also resolves itab (interface table) entries.
func (k *KnownInfo) AnalyzeTypes() error {
	slog.Info("Analyzing type descriptors...")

	md, err := k.Gore.Moduledata()
	if err != nil {
		return fmt.Errorf("type analysis moduledata: %w", err)
	}

	typesSection := md.Types()
	typesStart := typesSection.Address
	typesEnd := typesStart + typesSection.Length

	types, err := k.Gore.GetTypes()
	if err != nil {
		return fmt.Errorf("type analysis get types: %w", err)
	}

	if len(types) == 0 {
		slog.Info("No types found")
		return nil
	}

	// Build type address cache for itab resolution (local, not kept on struct).
	typeAddrCache := make(map[uint64]*gore.GoType, len(types))
	for _, t := range types {
		typeAddrCache[t.Addr] = t
	}

	// Sort types by address for size estimation.
	slices.SortFunc(types, func(a, b *gore.GoType) int {
		return cmp.Compare(a.Addr, b.Addr)
	})

	// Deduplicate by address (keep first occurrence after sort).
	deduped := make([]*gore.GoType, 0, len(types))
	var lastAddr uint64
	for _, t := range types {
		if len(deduped) > 0 && t.Addr == lastAddr {
			continue
		}
		lastAddr = t.Addr
		deduped = append(deduped, t)
	}

	// Filter to types within the types region.
	filtered := make([]*gore.GoType, 0, len(deduped))
	for _, t := range deduped {
		if t.Addr >= typesStart && t.Addr < typesEnd {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		slog.Info("No types within types region")
		return nil
	}

	var typesAttributed int

	for i, t := range filtered {
		// Legacy Go <1.7 parser path does not populate FlatSize; fall back to
		// the old gap-to-next estimate so those binaries still get sized.
		size := t.FlatSize
		if size == 0 {
			if i+1 < len(filtered) {
				size = filtered[i+1].Addr - t.Addr
			} else {
				size = typesEnd - t.Addr
			}
		}

		pkg := k.resolveTypePackage(t.PackagePath)
		symName := fmt.Sprintf("type:%s", t.Name)
		sym := entity.NewSymbol(symName, t.Addr, size, entity.AddrTypeData)

		ap := k.KnownAddr.InsertSymbol(sym, pkg)
		if ap == nil {
			continue
		}
		pkg.AddSymbol(sym, ap)
		typesAttributed++
	}

	auxAttributed, err := k.attributeTypeAuxRanges(typesEnd)
	if err != nil {
		return err
	}

	slog.Info("Type descriptors analyzed",
		"total", len(types),
		"inRegion", len(filtered),
		"attributed", typesAttributed,
		"auxAttributed", auxAttributed,
	)

	if err := k.analyzeItabs(md, typeAddrCache); err != nil {
		return fmt.Errorf("type analysis itabs: %w", err)
	}

	return nil
}

// resolveTypePackage returns the package a type-related symbol should be
// attributed to. Empty PackagePath means the type has no owning Go package
// (compiler-synthesized or shared), so it goes to the virtual runtime/types.
func (k *KnownInfo) resolveTypePackage(pkgPath string) *entity.Package {
	if pkgPath != "" {
		return k.resolvePackage(pkgPath, entity.PackageTypeUnknown)
	}
	return k.getOrCreateVirtualPackage("runtime/types", entity.PackageTypeGenerated)
}

// attributeTypeAuxRanges records every type-referenced byte span (names,
// method/field/imethod arrays, funcType arg arrays, shared package paths)
// reported by the gore parser as a data symbol under the owner type's package.
// Spans falling outside the moduledata types region are dropped because
// k.KnownAddr only tracks that region for type data.
func (k *KnownInfo) attributeTypeAuxRanges(typesEnd uint64) (int, error) {
	ranges, err := k.Gore.GetTypeAuxRanges()
	if err != nil {
		return 0, fmt.Errorf("type analysis aux ranges: %w", err)
	}

	var attributed int
	for _, r := range ranges {
		if r.Size == 0 || r.Addr+r.Size > typesEnd {
			continue
		}

		var ownerPath string
		if r.Owner != nil {
			ownerPath = r.Owner.PackagePath
		}
		pkg := k.resolveTypePackage(ownerPath)

		symName := auxSymbolName(r)
		sym := entity.NewSymbol(symName, r.Addr, r.Size, entity.AddrTypeData)

		ap := k.KnownAddr.InsertSymbol(sym, pkg)
		if ap == nil {
			continue
		}
		pkg.AddSymbol(sym, ap)
		attributed++
	}
	return attributed, nil
}

func auxSymbolName(r gore.TypeAuxRange) string {
	kind := auxKindTag(r.Kind)
	if r.Owner != nil && r.Owner.Name != "" {
		return fmt.Sprintf("type.%s:%s", kind, r.Owner.Name)
	}
	return fmt.Sprintf("type.%s:0x%x", kind, r.Addr)
}

func auxKindTag(k gore.TypeAuxKind) string {
	switch k {
	case gore.AuxName:
		return "name"
	case gore.AuxTag:
		return "tag"
	case gore.AuxPkgPath:
		return "pkgpath"
	case gore.AuxMethods:
		return "methods"
	case gore.AuxFields:
		return "fields"
	case gore.AuxIMethods:
		return "imethods"
	case gore.AuxFuncArgs:
		return "funcargs"
	default:
		return "aux"
	}
}

// itabStructSize returns the on-disk size of an itab given its interface type.
// Layout (runtime/iface.go): inter(ptr) + _type(ptr) + hash(4) + pad(4) +
// fun[n](ptr*n). Go always emits at least one fun slot even when n==0, so we
// floor at 1. Returns 0 when interType is nil so the caller can fall back.
func itabStructSize(interType *gore.GoType, ptrSz uint64) uint64 {
	if interType == nil {
		return 0
	}
	n := uint64(len(interType.Methods))
	if n == 0 {
		n = 1
	}
	return 2*ptrSz + 8 + n*ptrSz
}

// readPtr reads a pointer-sized value from data at the given byte offset.
func readPtr(data []byte, off uint64, ptrSize int, order binary.ByteOrder) uint64 {
	if ptrSize == 4 {
		return uint64(order.Uint32(data[off : off+4]))
	}
	return order.Uint64(data[off : off+8])
}

// analyzeItabs reads the itablinks pointer array from moduledata and attributes
// each itab struct to the package of its concrete type.
func (k *KnownInfo) analyzeItabs(md gore.Moduledata, typeAddrCache map[uint64]*gore.GoType) error {
	itabSection := md.ITabLinks()
	if itabSection.Length == 0 {
		return nil
	}

	ptrSize, order := ptrSizeAndOrder(k.Wrapper.GoArch())
	ptrSz := uint64(ptrSize)

	// ITabLinks Length is element count (Go slice len), not byte count.
	numItabs := itabSection.Length

	// Read the pointer array in one call.
	ptrData, err := k.Wrapper.ReadAddr(itabSection.Address, numItabs*ptrSz)
	if err != nil {
		return fmt.Errorf("reading itablinks pointer array: %w", err)
	}

	// Collect all non-zero itab addresses and find the contiguous region.
	// On PIE binaries, each pointer in the array is a fixup descriptor that
	// must be resolved to the actual virtual address.
	itabAddrs := make([]uint64, 0, numItabs)
	for i := range numItabs {
		fileAddr := itabSection.Address + i*ptrSz
		raw := readPtr(ptrData, i*ptrSz, ptrSize, order)
		addr := md.ResolvePointer(raw, fileAddr)
		if addr != 0 {
			itabAddrs = append(itabAddrs, addr)
		}
	}

	if len(itabAddrs) == 0 {
		return nil
	}

	slices.Sort(itabAddrs)

	// Bulk-read only the inter + _type header pair; the per-itab fun[] tail
	// has variable length so we size each one later via itabStructSize.
	headerSize := 2 * ptrSz
	regionStart := itabAddrs[0]
	regionEnd := itabAddrs[len(itabAddrs)-1] + headerSize
	regionData, err := k.Wrapper.ReadAddr(regionStart, regionEnd-regionStart)
	if err != nil {
		slog.Warn("Failed to bulk-read itab region, falling back to per-itab reads", "err", err)
		regionData = nil
	}

	minItabSize := 2*ptrSz + 8 + ptrSz

	readItabPtr := func(itabAddr, fieldOff uint64) (uint64, bool) {
		fieldFileAddr := itabAddr + fieldOff
		localOff := itabAddr - regionStart + fieldOff
		var raw uint64
		if regionData != nil && localOff+ptrSz <= uint64(len(regionData)) {
			raw = readPtr(regionData, localOff, ptrSize, order)
		} else {
			buf, err := k.Wrapper.ReadAddr(fieldFileAddr, ptrSz)
			if err != nil {
				return 0, false
			}
			raw = readPtr(buf, 0, ptrSize, order)
		}
		return md.ResolvePointer(raw, fieldFileAddr), true
	}

	var itabsAttributed int

	for idx, itabAddr := range itabAddrs {
		interAddr, okInter := readItabPtr(itabAddr, 0)
		typeAddr, okType := readItabPtr(itabAddr, ptrSz)
		if !okType {
			slog.Debug("Failed to read itab _type field", "itabAddr", itabAddr)
			continue
		}

		goType := typeAddrCache[typeAddr]
		var interType *gore.GoType
		if okInter {
			interType = typeAddrCache[interAddr]
		}

		itabSize := itabStructSize(interType, ptrSz)
		if itabSize == 0 {
			// Unknown interface: gap-to-next, last entry capped at minItabSize.
			if idx+1 < len(itabAddrs) {
				itabSize = itabAddrs[idx+1] - itabAddr
			} else {
				itabSize = minItabSize
			}
		}

		var ownerPath string
		if goType != nil {
			ownerPath = goType.PackagePath
		}
		pkg := k.resolveTypePackage(ownerPath)

		symName := fmt.Sprintf("itab:0x%x", itabAddr)
		switch {
		case goType != nil && interType != nil && interType.Name != "":
			symName = fmt.Sprintf("itab:%s,%s", interType.Name, goType.Name)
		case goType != nil:
			symName = fmt.Sprintf("itab:%s", goType.Name)
		}

		sym := entity.NewSymbol(symName, itabAddr, itabSize, entity.AddrTypeData)
		ap := k.KnownAddr.InsertSymbol(sym, pkg)
		if ap == nil {
			continue
		}
		pkg.AddSymbol(sym, ap)
		itabsAttributed++
	}

	slog.Info("Itabs analyzed",
		"total", numItabs,
		"attributed", itabsAttributed,
	)

	return nil
}
