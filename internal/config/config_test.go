package config

import (
	"os"
	"testing"
)

func TestExpandEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envName  string
		envValue string
		want     string
	}{
		{
			name:     "expand env var",
			input:    "${TEST_VAR}",
			envName:  "TEST_VAR",
			envValue: "test_value",
			want:     "test_value",
		},
		{
			name:  "no env var syntax",
			input: "plain_string",
			want:  "plain_string",
		},
		{
			name:     "empty env var",
			input:    "${EMPTY_VAR}",
			envName:  "EMPTY_VAR",
			envValue: "",
			want:     "",
		},
		{
			name:  "partial syntax - no closing brace",
			input: "${INCOMPLETE",
			want:  "${INCOMPLETE",
		},
		{
			name:  "partial syntax - no opening",
			input: "INCOMPLETE}",
			want:  "INCOMPLETE}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envName != "" {
				os.Setenv(tt.envName, tt.envValue)
				defer os.Unsetenv(tt.envName)
			}

			got := expandEnvVar(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVar(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid template with variables",
			template: "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			wantErr:  false,
		},
		{
			name:     "simple string without variables",
			template: "simple-name",
			wantErr:  false,
		},
		{
			name:     "invalid - unclosed action",
			template: "{{.Date}-{{.Service}}",
			wantErr:  true,
		},
		{
			name:     "invalid - unclosed brace",
			template: "{{.Date",
			wantErr:  true,
		},
		{
			name:     "empty template",
			template: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate(%q) error = %v, wantErr %v", tt.template, err, tt.wantErr)
			}
		})
	}
}

func TestBuildFullTemplate(t *testing.T) {
	tests := []struct {
		name           string
		servicePattern string
		want           string
	}{
		{
			name:           "with service variable",
			servicePattern: "{{.Service}}",
			want:           "{{.Date}}-{{.Service}}-{{.OriginalName}}",
		},
		{
			name:           "with static string",
			servicePattern: "MyCompany",
			want:           "{{.Date}}-MyCompany-{{.OriginalName}}",
		},
		{
			name:           "empty pattern",
			servicePattern: "",
			want:           "{{.Date}}--{{.OriginalName}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFullTemplate(tt.servicePattern)
			if got != tt.want {
				t.Errorf("BuildFullTemplate(%q) = %q, want %q", tt.servicePattern, got, tt.want)
			}
		})
	}
}

func TestProviderDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "anthropic",
			provider: "anthropic",
			want:     "Anthropic Claude API",
		},
		{
			name:     "empty provider",
			provider: "",
			want:     "未設定",
		},
		{
			name:     "unknown provider",
			provider: "custom",
			want:     "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AI: AIConfig{
					Provider: tt.provider,
				},
			}
			got := cfg.ProviderDisplayName()
			if got != tt.want {
				t.Errorf("ProviderDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetDefaultModel(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		existingModel string
		wantModel     string
		wantErr       bool
	}{
		{
			name:      "anthropic default",
			provider:  "anthropic",
			wantModel: "claude-sonnet-4-20250514",
		},
		{
			name:          "existing model not overwritten",
			provider:      "anthropic",
			existingModel: "custom-model",
			wantModel:     "custom-model",
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AI: AIConfig{
					Provider: tt.provider,
					Model:    tt.existingModel,
				},
			}

			err := cfg.setDefaultModel()
			if (err != nil) != tt.wantErr {
				t.Errorf("setDefaultModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg.AI.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cfg.AI.Model, tt.wantModel)
			}
		})
	}
}

func TestAutoDetectProvider(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		apiKey       string
		anthropicEnv string
		wantProvider string
		wantAPIKey   string
	}{
		{
			name:         "already configured",
			provider:     "anthropic",
			apiKey:       "sk-existing",
			wantProvider: "anthropic",
			wantAPIKey:   "sk-existing",
		},
		{
			name:         "detect from ANTHROPIC_API_KEY",
			anthropicEnv: "sk-ant-xxx",
			wantProvider: "anthropic",
			wantAPIKey:   "sk-ant-xxx",
		},
		{
			name:         "no api key found - just empty",
			wantProvider: "",
			wantAPIKey:   "",
		},
		{
			name:         "api key set but no provider defaults to anthropic",
			apiKey:       "sk-xxx",
			wantProvider: "anthropic",
			wantAPIKey:   "sk-xxx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			os.Unsetenv("ANTHROPIC_API_KEY")

			if tt.anthropicEnv != "" {
				os.Setenv("ANTHROPIC_API_KEY", tt.anthropicEnv)
				defer os.Unsetenv("ANTHROPIC_API_KEY")
			}

			cfg := &Config{
				AI: AIConfig{
					Provider: tt.provider,
					APIKey:   tt.apiKey,
				},
			}

			cfg.autoDetectProvider()

			if cfg.AI.Provider != tt.wantProvider {
				t.Errorf("Provider = %q, want %q", cfg.AI.Provider, tt.wantProvider)
			}
			if cfg.AI.APIKey != tt.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", cfg.AI.APIKey, tt.wantAPIKey)
			}
		})
	}
}
