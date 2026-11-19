package indexer

import "testing"

func TestGetSetMetadata(t *testing.T) {
	r := NewResultV2()

	// Initial check
	if _, ok := GetMetadata[string](r, "author"); ok {
		t.Errorf("expected no metadata for key 'author'")
	}

	// Set string
	SetMetadata(r, "author", "Albert Einstein")
	valStr, ok := GetMetadata[string](r, "author")
	if !ok || valStr != "Albert Einstein" {
		t.Errorf("expected 'Albert Einstein', got %q, ok=%v", valStr, ok)
	}

	// Type mismatch
	if _, ok := GetMetadata[int](r, "author"); ok {
		t.Errorf("expected type mismatch for int on 'author'")
	}

	// Set int
	SetMetadata(r, "year", 1905)
	valInt, ok := GetMetadata[int](r, "year")
	if !ok || valInt != 1905 {
		t.Errorf("expected 1905, got %d, ok=%v", valInt, ok)
	}
}
