package go_size_view

func increaseSectionSizeFromSymbol(sm *SectionMap) error {
	symtab := sm.SymTab
	for _, sym := range symtab.Symbols {
		if sym.SizeCounted {
			continue
		}

		err := sm.IncreaseKnown(sym.Addr, sym.Addr+uint64(sym.Size))
		if err != nil {
			return err
		}
	}
	return nil
}
