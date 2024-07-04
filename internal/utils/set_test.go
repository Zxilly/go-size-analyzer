package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSet(t *testing.T) {
	set := NewSet[int]()
	assert.NotNil(t, set)
	assert.Equal(t, 0, len(set))
}

func TestAdd(t *testing.T) {
	set := NewSet[int]()
	set.Add(1)
	assert.True(t, set.Contains(1))
}

func TestRemove(t *testing.T) {
	set := NewSet[int]()
	set.Add(1)
	set.Remove(1)
	assert.False(t, set.Contains(1))
}

func TestContains_ExistingItem(t *testing.T) {
	set := NewSet[int]()
	set.Add(1)
	assert.True(t, set.Contains(1))
}

func TestContains_NonExistingItem(t *testing.T) {
	set := NewSet[int]()
	assert.False(t, set.Contains(1))
}

func TestEquals_EqualSets(t *testing.T) {
	set1 := NewSet[int]()
	set1.Add(1)
	set1.Add(2)

	set2 := NewSet[int]()
	set2.Add(1)
	set2.Add(2)

	assert.True(t, set1.Equals(set2))
}

func TestEquals_NotEqualSets_DifferentLengths(t *testing.T) {
	set1 := NewSet[int]()
	set1.Add(1)

	set2 := NewSet[int]()
	set2.Add(1)
	set2.Add(2)

	assert.False(t, set1.Equals(set2))
}

func TestEquals_NotEqualSets_DifferentItems(t *testing.T) {
	set1 := NewSet[int]()
	set1.Add(1)

	set2 := NewSet[int]()
	set2.Add(2)

	assert.False(t, set1.Equals(set2))
}

func TestToSlice(t *testing.T) {
	set := NewSet[int]()
	set.Add(1)
	set.Add(2)

	slice := set.ToSlice()
	assert.Contains(t, slice, 1)
	assert.Contains(t, slice, 2)
	assert.Equal(t, 2, len(slice))
}
