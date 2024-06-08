package knowninfo

import (
	"debug/dwarf"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func (k *KnownInfo) TryLoadDwarf() bool {
	dwarf, err := k.Wrapper.DWARF()
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF: %v", err))
		return false
	}

	out, err := os.OpenFile("dwarf.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	r := dwarf.Reader()
	for {
		entry, err := r.Next()
		if entry == nil {
			break
		}
		if err != nil {
			slog.Warn(fmt.Sprintf("Failed to read DWARF entry: %#v", err))
			break
		}

		fmt.Fprintf(out, "DWARF entry: %v\n", entryPrettyPrinter(entry))
	}

	return true
}

func entryPrettyPrinter(entry *dwarf.Entry) string {
	s := new(strings.Builder)

	fmt.Fprintf(s, "Offset: %v\n", entry.Offset)
	fmt.Fprintf(s, "Tag: %v\n", entry.Tag.String())
	fmt.Fprintf(s, "Children: %v\n", entry.Children)
	for _, field := range entry.Field {
		fmt.Fprintf(s, "Field: %#v\n", field)
	}

	return s.String()
}
