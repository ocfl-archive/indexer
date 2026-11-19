package indexer

import (
	"slices"
	"testing"
)

func TestSet(t *testing.T) {
	s := NewSet("apple", "banana", "cherry")
	if !s.Contains("apple") {
		t.Errorf("expected set to contain 'apple'")
	}
	if s.Contains("orange") {
		t.Errorf("expected set not to contain 'orange'")
	}
	if !s.ContainsAll("apple", "cherry") {
		t.Errorf("expected set to contain all elements")
	}
	if s.ContainsAll("apple", "grape") {
		t.Errorf("expected ContainsAll to return false when an element is missing")
	}

	s.Add("orange", "lemon")
	if s.Len() != 5 {
		t.Errorf("expected length 5, got %d", s.Len())
	}

	s.Remove("banana")
	if s.Contains("banana") {
		t.Errorf("expected 'banana' to be removed")
	}

	s2 := NewSet("apple", "lemon", "grape")
	intersect := s.Intersect(s2)
	sortedIntersect := SortedElements(intersect)
	expectedIntersect := []string{"apple", "lemon"}
	if !slices.Equal(sortedIntersect, expectedIntersect) {
		t.Errorf("expected intersect %v, got %v", expectedIntersect, sortedIntersect)
	}

	union := s.Union(s2)
	if !union.ContainsAll("apple", "cherry", "orange", "lemon", "grape") {
		t.Errorf("expected union to contain all elements")
	}
}
