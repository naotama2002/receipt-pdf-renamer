package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Converter struct{}

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) ToImage(pdfPath string) ([]byte, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("pdftoppm not found: please install poppler (brew install poppler)")
	}

	tempDir, err := os.MkdirTemp("", "receipt-pdf-renamer-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	outputBase := filepath.Join(tempDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-singlefile", "-r", "150", pdfPath, outputBase)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to convert PDF to image: %w", err)
	}

	outputPath := outputBase + ".png"
	imageData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read converted image: %w", err)
	}

	return imageData, nil
}

func (c *Converter) IsAvailable() bool {
	_, err := exec.LookPath("pdftoppm")
	return err == nil
}
