package renamer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

type Renamer struct {
	template   *template.Template
	dateFormat string
}

type TemplateData struct {
	Date         string
	Service      string
	OriginalName string
}

func New(cfg *config.FormatConfig) (*Renamer, error) {
	tmpl, err := template.New("filename").Parse(cfg.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Renamer{
		template:   tmpl,
		dateFormat: cfg.DateFormat,
	}, nil
}

func (r *Renamer) UpdateTemplate(templateStr string) error {
	tmpl, err := template.New("filename").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	r.template = tmpl
	return nil
}

func (r *Renamer) GenerateName(originalPath string, info *ai.ReceiptInfo) (string, error) {
	originalName := filepath.Base(originalPath)
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)

	serviceName := sanitizeFilename(info.Service)

	data := TemplateData{
		Date:         info.Date,
		Service:      serviceName,
		OriginalName: nameWithoutExt,
	}

	var buf bytes.Buffer
	if err := r.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	newName := buf.String() + ext
	return newName, nil
}

func (r *Renamer) Rename(oldPath, newName string) error {
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newName)

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("destination file already exists: %s", newPath)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
		" ", "-",
	)
	result := replacer.Replace(s)

	result = strings.Trim(result, "-")

	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	return result
}
