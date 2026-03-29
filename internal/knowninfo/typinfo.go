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
// Failures are non-fatal: errors are logged and nil is returned.
func (k *KnownInfo) AnalyzeTypes() error {
	slog.Info("Analyzing type descriptors...")

	md, err := k.Gore.Moduledata()
	if err != nil {
		slog.Warn("Failed to get moduledata for type analysis", "err", err)
		return nil
	}

	typesSection := md.Types()
	typesStart := typesSection.Address
	typesEnd := typesStart + typesSection.Length

	types, err := k.Gore.GetTypes()
	if err != nil {
		slog.Warn("Failed to get types", "err", err)
		return nil
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
		// Estimate size as gap to next type; last type extends to region end.
		var size uint64
		if i+1 < len(filtered) {
			size = filtered[i+1].Addr - t.Addr
		} else {
			size = typesEnd - t.Addr
		}

		var pkg *entity.Package
		if t.PackagePath != "" {
			pkg = k.resolvePackage(t.PackagePath, entity.PackageTypeUnknown)
		} else {
			pkg = k.getOrCreateVirtualPackage("runtime/types", entity.PackageTypeGenerated)
		}

		symName := fmt.Sprintf("type:%s", t.Name)
		sym := entity.NewSymbol(symName, t.Addr, size, entity.AddrTypeData)

		ap := k.KnownAddr.InsertSymbol(sym, pkg)
		if ap == nil {
			continue
		}
		pkg.AddSymbol(sym, ap)
		typesAttributed++
	}

	slog.Info("Type descriptors analyzed",
		"total", len(types),
		"inRegion", len(filtered),
		"attributed", typesAttributed,
	)

	if err := k.analyzeItabs(md, typeAddrCache); err != nil {
		slog.Warn("Failed to analyze itabs", "err", err)
	}

	return nil
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
	itabAddrs := make([]uint64, 0, numItabs)
	for i := uint64(0); i < numItabs; i++ {
		addr := readPtr(ptrData, i*ptrSz, ptrSize, order)
		if addr != 0 {
			itabAddrs = append(itabAddrs, addr)
		}
	}

	if len(itabAddrs) == 0 {
		return nil
	}

	slices.Sort(itabAddrs)

	// Bulk-read the contiguous region covering all itab structs.
	// Each itab needs at least 2*ptrSize bytes (inter + _type fields).
	minItabFieldsSize := 2 * ptrSz
	regionStart := itabAddrs[0]
	regionEnd := itabAddrs[len(itabAddrs)-1] + minItabFieldsSize
	regionData, err := k.Wrapper.ReadAddr(regionStart, regionEnd-regionStart)
	if err != nil {
		slog.Warn("Failed to bulk-read itab region, falling back to per-itab reads", "err", err)
		regionData = nil
	}

	// Minimum itab size for the last entry: inter + _type + hash/pad + 1 method slot.
	minItabSize := 2*ptrSz + 8 + ptrSz

	var itabsAttributed int

	for idx, itabAddr := range itabAddrs {
		// Use gap to next itab for size; last itab uses conservative minimum.
		var itabSize uint64
		if idx+1 < len(itabAddrs) {
			itabSize = itabAddrs[idx+1] - itabAddr
		} else {
			itabSize = minItabSize
		}
		// Read _type pointer (at offset ptrSize within the itab struct).
		var typeAddr uint64
		localOff := itabAddr - regionStart + ptrSz
		if regionData != nil && localOff+ptrSz <= uint64(len(regionData)) {
			typeAddr = readPtr(regionData, localOff, ptrSize, order)
		} else {
			// Fallback: individual read for out-of-range itabs.
			typeFieldData, err := k.Wrapper.ReadAddr(itabAddr+ptrSz, ptrSz)
			if err != nil {
				slog.Debug("Failed to read itab _type field", "itabAddr", itabAddr, "err", err)
				continue
			}
			typeAddr = readPtr(typeFieldData, 0, ptrSize, order)
		}

		goType, ok := typeAddrCache[typeAddr]

		var pkg *entity.Package
		if ok && goType.PackagePath != "" {
			pkg = k.resolvePackage(goType.PackagePath, entity.PackageTypeUnknown)
		} else {
			pkg = k.getOrCreateVirtualPackage("runtime/types", entity.PackageTypeGenerated)
		}

		symName := fmt.Sprintf("itab:0x%x", itabAddr)
		if ok {
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
