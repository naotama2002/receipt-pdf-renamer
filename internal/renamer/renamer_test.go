package renamer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special characters",
			input: "Cursor",
			want:  "Cursor",
		},
		{
			name:  "spaces to hyphens",
			input: "GitHub Copilot",
			want:  "GitHub-Copilot",
		},
		{
			name:  "slashes to hyphens",
			input: "AWS/EC2",
			want:  "AWS-EC2",
		},
		{
			name:  "colons to hyphens",
			input: "Service:Name",
			want:  "Service-Name",
		},
		{
			name:  "remove asterisks",
			input: "Service*Name",
			want:  "ServiceName",
		},
		{
			name:  "remove question marks",
			input: "Service?Name",
			want:  "ServiceName",
		},
		{
			name:  "remove quotes",
			input: `Service"Name`,
			want:  "ServiceName",
		},
		{
			name:  "remove angle brackets",
			input: "Service<Name>",
			want:  "ServiceName",
		},
		{
			name:  "remove pipes",
			input: "Service|Name",
			want:  "ServiceName",
		},
		{
			name:  "multiple spaces become single hyphen",
			input: "Service   Name",
			want:  "Service-Name",
		},
		{
			name:  "trim leading/trailing hyphens",
			input: " Service Name ",
			want:  "Service-Name",
		},
		{
			name:  "complex combination",
			input: " AWS / EC2 : Instance ",
			want:  "AWS-EC2-Instance",
		},
		{
			name:  "backslash to hyphen",
			input: "Path\\To\\Service",
			want:  "Path-To-Service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateName(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		originalPath string
		info         *ai.ReceiptInfo
		want         string
		wantErr      bool
	}{
		{
			name:         "standard template",
			template:     "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			originalPath: "/path/to/Receipt-001.pdf",
			info:         &ai.ReceiptInfo{Date: "20250115", Service: "Cursor"},
			want:         "20250115-Cursor-Receipt-001.pdf",
		},
		{
			name:         "service with spaces gets sanitized",
			template:     "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			originalPath: "/path/to/Invoice.pdf",
			info:         &ai.ReceiptInfo{Date: "20250120", Service: "GitHub Copilot"},
			want:         "20250120-GitHub-Copilot-Invoice.pdf",
		},
		{
			name:         "static service pattern",
			template:     "{{.Date}}-MyCompany-{{.OriginalName}}",
			originalPath: "/path/to/receipt.pdf",
			info:         &ai.ReceiptInfo{Date: "20250101", Service: "Ignored"},
			want:         "20250101-MyCompany-receipt.pdf",
		},
		{
			name:         "preserves extension",
			template:     "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			originalPath: "/path/to/file.PDF",
			info:         &ai.ReceiptInfo{Date: "20250101", Service: "Test"},
			want:         "20250101-Test-file.PDF",
		},
		{
			name:         "handles file without extension",
			template:     "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			originalPath: "/path/to/noextension",
			info:         &ai.ReceiptInfo{Date: "20250101", Service: "Test"},
			want:         "20250101-Test-noextension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(&config.FormatConfig{
				Template:   tt.template,
				DateFormat: "20060102",
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			got, err := r.GenerateName(tt.originalPath, tt.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateTemplate(t *testing.T) {
	r, err := New(&config.FormatConfig{
		Template:   "{{.Date}}-{{.OriginalName}}",
		DateFormat: "20060102",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// 初期テンプレートでの生成
	info := &ai.ReceiptInfo{Date: "20250115", Service: "Cursor"}
	got, _ := r.GenerateName("/path/to/test.pdf", info)
	if got != "20250115-test.pdf" {
		t.Errorf("Initial template: got %q, want %q", got, "20250115-test.pdf")
	}

	// テンプレート更新
	err = r.UpdateTemplate("{{.Date}}-{{.Service}}-{{.OriginalName}}")
	if err != nil {
		t.Fatalf("UpdateTemplate() error = %v", err)
	}

	// 更新後のテンプレートでの生成
	got, _ = r.GenerateName("/path/to/test.pdf", info)
	if got != "20250115-Cursor-test.pdf" {
		t.Errorf("Updated template: got %q, want %q", got, "20250115-Cursor-test.pdf")
	}
}

func TestUpdateTemplate_InvalidTemplate(t *testing.T) {
	r, _ := New(&config.FormatConfig{
		Template:   "{{.Date}}",
		DateFormat: "20060102",
	})

	err := r.UpdateTemplate("{{.Invalid")
	if err == nil {
		t.Error("UpdateTemplate() with invalid template should return error")
	}
}

func TestRename(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "renamer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	r, _ := New(&config.FormatConfig{
		Template:   "{{.Date}}-{{.Service}}-{{.OriginalName}}",
		DateFormat: "20060102",
	})

	t.Run("successful rename", func(t *testing.T) {
		// テスト用ファイルを作成
		oldPath := filepath.Join(tmpDir, "original.pdf")
		if err := os.WriteFile(oldPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		newName := "renamed.pdf"
		err := r.Rename(oldPath, newName)
		if err != nil {
			t.Errorf("Rename() error = %v", err)
		}

		// 新しいファイルが存在することを確認
		newPath := filepath.Join(tmpDir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Error("Renamed file does not exist")
		}

		// 古いファイルが存在しないことを確認
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Error("Original file should not exist after rename")
		}
	})

	t.Run("destination already exists", func(t *testing.T) {
		// 2つのファイルを作成
		oldPath := filepath.Join(tmpDir, "source.pdf")
		existingPath := filepath.Join(tmpDir, "existing.pdf")
		if err := os.WriteFile(oldPath, []byte("source"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		if err := os.WriteFile(existingPath, []byte("existing"), 0644); err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		err := r.Rename(oldPath, "existing.pdf")
		if err == nil {
			t.Error("Rename() should return error when destination exists")
		}
	})
}
