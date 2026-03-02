package pdf

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Extractor はpdftotext を使ってPDFからテキストを抽出する
type Extractor struct {
	path string
}

// NewExtractor は Extractor を生成する
// path が空の場合は exec.LookPath で pdftotext を探す
func NewExtractor(path string) *Extractor {
	if path == "" {
		if found, err := exec.LookPath("pdftotext"); err == nil {
			path = found
		}
	}
	return &Extractor{path: path}
}

// IsAvailable は pdftotext コマンドが利用可能かチェックする
func (e *Extractor) IsAvailable() bool {
	return e.path != ""
}

// Path は設定されている pdftotext のパスを返す
func (e *Extractor) Path() string {
	return e.path
}

// ExtractText は pdftotext -layout でPDFからテキストを抽出する
func (e *Extractor) ExtractText(pdfPath string) (string, error) {
	if e.path == "" {
		return "", fmt.Errorf("pdftotext not found: install poppler (macOS: brew install poppler)")
	}

	cmd := exec.Command(e.path, "-layout", pdfPath, "-")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, stderr.String())
	}

	text := stdout.String()
	if text == "" {
		return "", fmt.Errorf("pdftotext returned empty text for %s", pdfPath)
	}

	return text, nil
}
