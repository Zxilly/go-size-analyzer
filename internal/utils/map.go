package utils

import (
	"cmp"
	"slices"

	"golang.org/x/exp/maps"
)

func SortedKeys[T cmp.Ordered, U any](m map[T]U) []T {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}
