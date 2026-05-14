// Package collection provides generic slice and map utilities built on top of
// the standard library's slices and maps packages. All functions are pure and
// allocation-safe — they allocate only what is needed for the output.
package collection

import (
	"cmp"
	"maps"
	"slices"
)

// --- Set ---

// Set is a generic, unordered set backed by a map.
type Set[T comparable] map[T]struct{}

// NewSet creates a Set pre-populated with vals.
func NewSet[T comparable](vals ...T) Set[T] {
	s := make(Set[T], len(vals))
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// Add inserts item into the set and returns the set for chaining.
func (s Set[T]) Add(item T) Set[T] {
	s[item] = struct{}{}
	return s
}

// Remove deletes item from the set. No-op if item is absent.
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// Contains reports whether item is in the set.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

// Len returns the number of elements in the set.
func (s Set[T]) Len() int { return len(s) }

// ToSlice returns the set elements as a slice in unspecified order.
func (s Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s))
	for item := range s {
		result = append(result, item)
	}
	return result
}

// Union returns a new set containing all elements from s and other.
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := make(Set[T], len(s)+len(other))
	for k := range s {
		result[k] = struct{}{}
	}
	for k := range other {
		result[k] = struct{}{}
	}
	return result
}

// Intersect returns a new set containing only elements present in both s and other.
func (s Set[T]) Intersect(other Set[T]) Set[T] {
	result := make(Set[T])
	for k := range s {
		if other.Contains(k) {
			result[k] = struct{}{}
		}
	}
	return result
}

// --- Map helpers ---

// ListToGetterMap converts a slice to a map using a key-value extractor.
func ListToGetterMap[K comparable, T, V any](s []T, getKeyValue func(e T) (k K, v V)) map[K]V {
	res := make(map[K]V, len(s))
	for _, e := range s {
		key, value := getKeyValue(e)
		res[key] = value
	}
	return res
}

// ListToMap converts a slice to a map using a key extractor.
func ListToMap[T any, K comparable](vals []T, getKey func(T) K) map[K]T {
	res := make(map[K]T, len(vals))
	for _, val := range vals {
		res[getKey(val)] = val
	}
	return res
}

// MapToListKey returns all keys of a map as a slice.
func MapToListKey[K comparable, V any](m map[K]V) []K {
	return slices.AppendSeq(make([]K, 0, len(m)), maps.Keys(m))
}

// MapToListValue returns all values of a map as a slice.
func MapToListValue[K comparable, V any](m map[K]V) []V {
	return slices.AppendSeq(make([]V, 0, len(m)), maps.Values(m))
}

// --- Slice helpers ---

// IsInSlice reports whether x is present in ss.
func IsInSlice[T comparable](x T, ss []T) bool {
	return slices.Contains(ss, x)
}

// DeduplicateSlice removes duplicates while preserving order.
func DeduplicateSlice[E comparable](s []E) []E {
	seen := make(Set[E], len(s))
	result := make([]E, 0, len(s))
	for _, e := range s {
		if !seen.Contains(e) {
			seen.Add(e)
			result = append(result, e)
		}
	}
	return result
}

// DeduplicateSliceAndRemove removes duplicates and excludes specified elements, preserving order.
func DeduplicateSliceAndRemove[E comparable](s []E, toRemove ...E) []E {
	exclude := NewSet(toRemove...)
	seen := make(Set[E], len(s))
	result := make([]E, 0, len(s))
	for _, e := range s {
		if exclude.Contains(e) {
			continue
		}
		if !seen.Contains(e) {
			seen.Add(e)
			result = append(result, e)
		}
	}
	return result
}

// DeduplicateAndSortSlice removes duplicates and sorts the result.
func DeduplicateAndSortSlice[S ~[]E, E cmp.Ordered](s S) S {
	slices.SortFunc(s, cmp.Compare)
	return slices.CompactFunc(s, func(e E, e2 E) bool {
		return cmp.Compare(e, e2) == 0
	})
}

// DeduplicateSliceOfObject removes duplicate objects by key, preserving first-seen order.
func DeduplicateSliceOfObject[E any, K comparable](s []E, getKey func(e E) K) []E {
	seen := make(map[K]struct{}, len(s))
	result := make([]E, 0, len(s))
	for _, e := range s {
		k := getKey(e)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			result = append(result, e)
		}
	}
	return result
}

// DeduplicateAndSortSliceOfObject removes duplicates by key and sorts by key.
func DeduplicateAndSortSliceOfObject[E any, K cmp.Ordered](s []E, getKey func(e E) K) []E {
	dedup := make(map[K]E, len(s))
	for _, e := range s {
		dedup[getKey(e)] = e
	}
	keys := slices.AppendSeq(make([]K, 0, len(dedup)), maps.Keys(dedup))
	slices.SortFunc(keys, cmp.Compare)
	return MapSlice(keys, func(k K) E { return dedup[k] })
}

// FilterSlice returns elements for which keep returns true.
func FilterSlice[T any](s []T, keep func(T) bool) []T {
	result := make([]T, 0)
	for _, e := range s {
		if keep(e) {
			result = append(result, e)
		}
	}
	return result
}

// MapSlice transforms each element of s using mapper.
func MapSlice[T, R any](s []T, mapper func(T) R) []R {
	result := make([]R, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}

// MapAndFilterSlice transforms elements and keeps only those where ok is true.
func MapAndFilterSlice[T, R any](s []T, mapper func(T) (R, bool)) []R {
	result := make([]R, 0)
	for i := range s {
		if item, ok := mapper(s[i]); ok {
			result = append(result, item)
		}
	}
	return result
}

// ReduceSlice folds s into a single value using reducer.
func ReduceSlice[T, R any](s []T, acc R, reducer func(e T, acc R) R) R {
	for i := range s {
		acc = reducer(s[i], acc)
	}
	return acc
}

// ReduceSliceWithCondition folds s into a value, stopping early when reducer returns false.
func ReduceSliceWithCondition[T, R any](s []T, acc R, reducer func(e T, acc R) (r R, keepContinuing bool)) R {
	for i := range s {
		var keepContinuing bool
		acc, keepContinuing = reducer(s[i], acc)
		if !keepContinuing {
			return acc
		}
	}
	return acc
}

// GetCommonElements returns elements present in all provided lists.
func GetCommonElements[T comparable](lists ...[]T) []T {
	switch len(lists) {
	case 0:
		return nil
	case 1:
		return lists[0]
	default:
		result := NewSet(lists[0]...)
		for _, list := range lists[1:] {
			if len(list) == 0 {
				return []T{}
			}
			result = result.Intersect(NewSet(list...))
		}
		return result.ToSlice()
	}
}
