package indexer

import (
	"cmp"
	"slices"
)

// Set is a generic set data structure for comparable elements.
type Set[T comparable] map[T]struct{}

// NewSet creates a new Set containing the given elements.
func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// Add inserts one or more elements into the set.
func (s Set[T]) Add(items ...T) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

// Remove removes one or more elements from the set.
func (s Set[T]) Remove(items ...T) {
	for _, item := range items {
		delete(s, item)
	}
}

// Contains checks if the set contains the specified element in O(1) time.
func (s Set[T]) Contains(item T) bool {
	if s == nil {
		return false
	}
	_, ok := s[item]
	return ok
}

// ContainsAll checks if all specified items are present in the set.
func (s Set[T]) ContainsAll(items ...T) bool {
	if s == nil {
		return len(items) == 0
	}
	for _, item := range items {
		if _, ok := s[item]; !ok {
			return false
		}
	}
	return true
}

// Intersect returns a new Set containing only elements present in both sets.
func (s Set[T]) Intersect(other Set[T]) Set[T] {
	res := make(Set[T])
	if s == nil || other == nil {
		return res
	}
	// Iterate through the smaller set to maximize efficiency
	if len(s) > len(other) {
		for k := range other {
			if _, ok := s[k]; ok {
				res[k] = struct{}{}
			}
		}
	} else {
		for k := range s {
			if _, ok := other[k]; ok {
				res[k] = struct{}{}
			}
		}
	}
	return res
}

// Union returns a new Set containing all unique elements from both sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	res := make(Set[T], len(s)+len(other))
	for k := range s {
		res[k] = struct{}{}
	}
	for k := range other {
		res[k] = struct{}{}
	}
	return res
}

// Elements returns a slice of all elements in the set.
func (s Set[T]) Elements() []T {
	if s == nil {
		return nil
	}
	res := make([]T, 0, len(s))
	for k := range s {
		res = append(res, k)
	}
	return res
}

// Len returns the number of elements in the set.
func (s Set[T]) Len() int {
	return len(s)
}

// SortedElements returns the elements of an ordered set sorted in ascending order.
func SortedElements[T cmp.Ordered](s Set[T]) []T {
	elems := s.Elements()
	slices.Sort(elems)
	return elems
}

// SortedElementsFunc returns elements sorted using a custom comparison function.
func (s Set[T]) SortedElementsFunc(cmpFunc func(a, b T) int) []T {
	elems := s.Elements()
	slices.SortFunc(elems, cmpFunc)
	return elems
}
