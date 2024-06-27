package utils

import (
	"cmp"
	"maps"
	"slices"
)

func SortedKeys[T cmp.Ordered, U any](m map[T]U) []T {
	keys := Collect(maps.Keys(m))
	slices.Sort(keys)
	return keys
}
