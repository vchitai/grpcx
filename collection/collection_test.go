package collection_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vchitai/grpcx/collection"
)

// --- Set ---

func TestNewSet(t *testing.T) {
	s := collection.NewSet(1, 2, 3, 2)
	assert.Equal(t, 3, s.Len())
	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(3))
	assert.False(t, s.Contains(9))
}

func TestSet_Add(t *testing.T) {
	s := collection.NewSet("a").Add("b").Add("a")
	assert.Equal(t, 2, s.Len())
}

func TestSet_Remove(t *testing.T) {
	s := collection.NewSet(1, 2, 3)
	s.Remove(2)
	assert.False(t, s.Contains(2))
	assert.Equal(t, 2, s.Len())
	s.Remove(99) // no-op
}

func TestSet_ToSlice(t *testing.T) {
	s := collection.NewSet("x", "y")
	assert.ElementsMatch(t, []string{"x", "y"}, s.ToSlice())
}

func TestSet_Union(t *testing.T) {
	a := collection.NewSet(1, 2, 3)
	b := collection.NewSet(3, 4, 5)
	u := a.Union(b)
	assert.Equal(t, 5, u.Len())
	assert.True(t, u.Contains(1))
	assert.True(t, u.Contains(5))
}

func TestSet_Intersect(t *testing.T) {
	a := collection.NewSet(1, 2, 3)
	b := collection.NewSet(2, 3, 4)
	i := a.Intersect(b)
	assert.Equal(t, 2, i.Len())
	assert.True(t, i.Contains(2))
	assert.True(t, i.Contains(3))
	assert.False(t, i.Contains(1))
}

// --- Map helpers ---

func TestListToMap(t *testing.T) {
	m := collection.ListToMap([]string{"a", "b", "c"}, func(s string) string { return s })
	assert.Equal(t, "a", m["a"])
	assert.Equal(t, "c", m["c"])
}

func TestMapSlice(t *testing.T) {
	out := collection.MapSlice([]int{1, 2, 3}, func(n int) int { return n * 2 })
	assert.Equal(t, []int{2, 4, 6}, out)
}

// --- Slice helpers ---

func TestFilterSlice(t *testing.T) {
	out := collection.FilterSlice([]int{1, 2, 3, 4}, func(n int) bool { return n%2 == 0 })
	assert.Equal(t, []int{2, 4}, out)
}

func TestMapAndFilterSlice(t *testing.T) {
	out := collection.MapAndFilterSlice([]int{1, 2, 3, 4}, func(n int) (int, bool) {
		return n * 10, n%2 == 0
	})
	assert.Equal(t, []int{20, 40}, out)
}

func TestDeduplicateSlice(t *testing.T) {
	out := collection.DeduplicateSlice([]int{1, 2, 2, 3, 1})
	assert.Equal(t, []int{1, 2, 3}, out)
}

func TestDeduplicateSliceAndRemove(t *testing.T) {
	out := collection.DeduplicateSliceAndRemove([]int{1, 2, 2, 3, 4}, 2, 4)
	assert.Equal(t, []int{1, 3}, out)
}

func TestDeduplicateAndSortSlice(t *testing.T) {
	out := collection.DeduplicateAndSortSlice([]int{3, 1, 2, 1, 3})
	assert.Equal(t, []int{1, 2, 3}, out)
}

func TestReduceSlice(t *testing.T) {
	sum := collection.ReduceSlice([]int{1, 2, 3, 4}, 0, func(e, acc int) int { return acc + e })
	assert.Equal(t, 10, sum)
}

func TestReduceSliceWithCondition(t *testing.T) {
	sum := collection.ReduceSliceWithCondition([]int{1, 2, 3, 4, 5}, 0, func(e, acc int) (int, bool) {
		acc += e
		return acc, acc <= 5
	})
	assert.Equal(t, 6, sum)
}

func TestGetCommonElements(t *testing.T) {
	out := collection.GetCommonElements([]int{1, 2, 3}, []int{2, 3, 4}, []int{3, 4, 5})
	assert.Equal(t, []int{3}, out)
}

func TestGetCommonElements_empty(t *testing.T) {
	out := collection.GetCommonElements[int]()
	assert.Nil(t, out)
}

func TestGetCommonElements_emptyList(t *testing.T) {
	out := collection.GetCommonElements([]int{1, 2}, []int{})
	assert.Empty(t, out)
}

func TestIsInSlice(t *testing.T) {
	assert.True(t, collection.IsInSlice(2, []int{1, 2, 3}))
	assert.False(t, collection.IsInSlice(9, []int{1, 2, 3}))
}

func TestMapToListKey(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	assert.ElementsMatch(t, []string{"a", "b"}, collection.MapToListKey(m))
}

func TestMapToListValue(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	assert.ElementsMatch(t, []int{1, 2}, collection.MapToListValue(m))
}
