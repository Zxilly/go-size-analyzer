package dwarf

import (
	"debug/dwarf"
	"fmt"
)

type fieldPattern struct {
	name string
	typ  string
}

func checkField(typ *dwarf.StructType, fields ...fieldPattern) error {
	if len(typ.Field) != len(fields) {
		return fmt.Errorf("%s struct has %d fields", typ.StructName, len(typ.Field))
	}

	for i, field := range fields {
		if typ.Field[i].Name != field.name {
			return fmt.Errorf("field %d name is %s, expect %s", i, typ.Field[i].Name, field.name)
		}

		if typ.Field[i].Type.String() != field.typ {
			return fmt.Errorf("field %d type is %s, expect %s", i, typ.Field[i].Type.String(), field.typ)
		}
	}

	return nil
}
