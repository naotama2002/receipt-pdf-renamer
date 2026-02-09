package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGet_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "nonexistent.json"))

	got := h.Get()

	if len(got) != 0 {
		t.Errorf("Get() = %v, want empty slice", got)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(filePath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := NewWithPath(filePath)
	got := h.Get()

	if len(got) != 0 {
		t.Errorf("Get() = %v, want empty slice for invalid JSON", got)
	}
}

func TestAdd_NewPattern(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	err := h.Add("{{.Service}}")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got := h.Get()
	want := []string{"{{.Service}}"}

	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestAdd_EmptyPattern(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	err := h.Add("")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got := h.Get()
	if len(got) != 0 {
		t.Errorf("Get() = %v, want empty slice for empty pattern", got)
	}
}

func TestAdd_MostRecentFirst(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	for _, p := range []string{"first", "second", "third"} {
		if err := h.Add(p); err != nil {
			t.Fatalf("Add(%q) error = %v", p, err)
		}
	}

	got := h.Get()
	want := []string{"third", "second", "first"}

	if len(got) != len(want) {
		t.Fatalf("Get() length = %d, want %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("Get()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestAdd_DuplicateMovesToFront(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	for _, p := range []string{"first", "second", "third", "first"} {
		if err := h.Add(p); err != nil {
			t.Fatalf("Add(%q) error = %v", p, err)
		}
	}

	got := h.Get()
	want := []string{"first", "third", "second"}

	if len(got) != len(want) {
		t.Fatalf("Get() length = %d, want %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("Get()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestAdd_MaxItemsLimit(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	// Add more than MaxItems
	for i := 0; i < MaxItems+5; i++ {
		if err := h.Add(string(rune('a' + i))); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	got := h.Get()

	if len(got) != MaxItems {
		t.Errorf("Get() length = %d, want %d (MaxItems)", len(got), MaxItems)
	}

	// Most recent should be first
	expected := string(rune('a' + MaxItems + 4)) // last added
	if got[0] != expected {
		t.Errorf("Get()[0] = %q, want %q (most recent)", got[0], expected)
	}
}

func TestAdd_MaxItemsWithDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	h := NewWithPath(filepath.Join(tmpDir, "history.json"))

	// Add MaxItems patterns
	for i := 0; i < MaxItems; i++ {
		if err := h.Add(string(rune('a' + i))); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Add duplicate of oldest item - should move to front, not increase count
	oldest := "a"
	if err := h.Add(oldest); err != nil {
		t.Fatalf("Add(%q) error = %v", oldest, err)
	}

	got := h.Get()

	if len(got) != MaxItems {
		t.Errorf("Get() length = %d, want %d", len(got), MaxItems)
	}

	if got[0] != oldest {
		t.Errorf("Get()[0] = %q, want %q (duplicate moved to front)", got[0], oldest)
	}
}
