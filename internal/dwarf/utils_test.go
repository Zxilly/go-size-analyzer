package dwarf

import (
	"debug/dwarf"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckFieldValidatesStructFieldsCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		typ      *dwarf.StructType
		fields   []fieldPattern
		expected string
	}{
		{
			name: "ValidStructWithMatchingFields",
			typ: &dwarf.StructType{
				StructName: "TestStruct",
				Field: []*dwarf.StructField{
					{Name: "Field1", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int"}}},
					{Name: "Field2", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "string"}}},
				},
			},
			fields: []fieldPattern{
				{name: "Field1", typ: "int"},
				{name: "Field2", typ: "string"},
			},
			expected: "",
		},
		{
			name: "StructWithMismatchedFieldCount",
			typ: &dwarf.StructType{
				StructName: "TestStruct",
				Field: []*dwarf.StructField{
					{Name: "Field1", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int"}}},
				},
			},
			fields: []fieldPattern{
				{name: "Field1", typ: "int"},
				{name: "Field2", typ: "string"},
			},
			expected: "TestStruct struct has 1 fields",
		},
		{
			name: "StructWithMismatchedFieldName",
			typ: &dwarf.StructType{
				StructName: "TestStruct",
				Field: []*dwarf.StructField{
					{Name: "Field1", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int"}}},
					{Name: "Field2", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "string"}}},
				},
			},
			fields: []fieldPattern{
				{name: "Field1", typ: "int"},
				{name: "Field3", typ: "string"},
			},
			expected: "field 1 name is Field2, expect Field3",
		},
		{
			name: "StructWithMismatchedFieldType",
			typ: &dwarf.StructType{
				StructName: "TestStruct",
				Field: []*dwarf.StructField{
					{Name: "Field1", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int"}}},
					{Name: "Field2", Type: &dwarf.BasicType{CommonType: dwarf.CommonType{Name: "string"}}},
				},
			},
			fields: []fieldPattern{
				{name: "Field1", typ: "int"},
				{name: "Field2", typ: "bool"},
			},
			expected: "field 1 type is string, expect bool",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkField(test.typ, test.fields...)
			if err != nil {
				require.Equal(t, test.expected, err.Error())
			}
			if err == nil && test.expected != "" {
				require.Empty(t, test.expected)
			}
		})
	}
}
