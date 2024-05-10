package utils_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSortedKeysReturnsSortedKeysForIntegerMap(t *testing.T) {
	m := map[int]string{3: "three", 1: "one", 2: "two"}
	expected := []int{1, 2, 3}
	result := utils.SortedKeys(m)
	assert.Equal(t, expected, result)
}

func TestSortedKeysReturnsSortedKeysForStringMap(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1, "c": 3}
	expected := []string{"a", "b", "c"}
	result := utils.SortedKeys(m)
	assert.Equal(t, expected, result)
}

func TestSortedKeysReturnsEmptySliceForEmptyMap(t *testing.T) {
	m := map[int]string{}
	expected := make([]int, 0)
	result := utils.SortedKeys(m)
	assert.Equal(t, expected, result)
}
