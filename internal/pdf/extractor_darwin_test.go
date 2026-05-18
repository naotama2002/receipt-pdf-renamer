//go:build darwin

package pdf

import (
	"testing"
)

func TestNewExtractorDarwin(t *testing.T) {
	extractor := NewExtractor("")

	// macOS では path が空でも IsAvailable は true（osascript を使うため）
	if !extractor.IsAvailable() {
		t.Error("IsAvailable() should always be true on macOS")
	}
}

func TestNewExtractorDarwinWithPath(t *testing.T) {
	extractor := NewExtractor("/some/path")

	if extractor.Path() != "/some/path" {
		t.Errorf("Path() = %q, want %q", extractor.Path(), "/some/path")
	}

	if !extractor.IsAvailable() {
		t.Error("IsAvailable() should always be true on macOS")
	}
}
