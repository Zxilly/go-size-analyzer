package dwarf

import (
	"debug/dwarf"
	"fmt"
)

func checkField(typ *dwarf.StructType, fields ...string) error {
	if len(typ.Field) != len(fields) {
		return fmt.Errorf("%s struct has %d fields", typ.StructName, len(typ.Field))
	}

	for i, field := range fields {
		if typ.Field[i].Name != field {
			return fmt.Errorf("%s struct has wrong field name", typ.StructName)
		}
	}

	return nil
}
