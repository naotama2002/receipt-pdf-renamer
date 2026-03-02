package pdf

import (
	"os/exec"
	"testing"
)

func TestNewExtractorWithEmptyPath(t *testing.T) {
	extractor := NewExtractor("")

	// LookPath で見つかるかどうかと一致するはず
	expectedPath, _ := exec.LookPath("pdftotext")

	if extractor.Path() != expectedPath {
		t.Errorf("NewExtractor(\"\").Path() = %q, want %q", extractor.Path(), expectedPath)
	}

	if expectedPath != "" && !extractor.IsAvailable() {
		t.Error("IsAvailable() should be true when pdftotext is in PATH")
	}
	if expectedPath == "" && extractor.IsAvailable() {
		t.Error("IsAvailable() should be false when pdftotext is not in PATH")
	}
}

func TestNewExtractorWithExplicitPath(t *testing.T) {
	extractor := NewExtractor("/usr/bin/pdftotext")

	if extractor.Path() != "/usr/bin/pdftotext" {
		t.Errorf("NewExtractor(\"/usr/bin/pdftotext\").Path() = %q, want %q", extractor.Path(), "/usr/bin/pdftotext")
	}

	if !extractor.IsAvailable() {
		t.Error("IsAvailable() should be true when path is explicitly set")
	}
}

func TestNewExtractorNotAvailable(t *testing.T) {
	// 存在しないパスを指定してもそのまま設定される（実行時にエラー）
	extractor := NewExtractor("/nonexistent/path/pdftotext")

	if !extractor.IsAvailable() {
		t.Error("IsAvailable() returns true for any non-empty path (validation is at execution time)")
	}
}

func TestExtractTextWithoutPath(t *testing.T) {
	extractor := &Extractor{path: ""}

	_, err := extractor.ExtractText("dummy.pdf")
	if err == nil {
		t.Error("ExtractText should fail when path is empty")
	}
}
