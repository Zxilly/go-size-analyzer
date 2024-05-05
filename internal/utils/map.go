package utils

import (
	"cmp"
	"golang.org/x/exp/maps"
	"slices"
)

func SortedKeys[T cmp.Ordered, U any](m map[T]U) []T {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}
