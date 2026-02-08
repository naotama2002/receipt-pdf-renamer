package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AI     AIConfig     `yaml:"ai"`
	Cache  CacheConfig  `yaml:"cache"`
	Format FormatConfig `yaml:"format"`
}

type AIConfig struct {
	Provider   string `yaml:"provider,omitempty"`
	BaseURL    string `yaml:"base_url,omitempty"`
	APIKey     string `yaml:"api_key,omitempty"`
	Model      string `yaml:"model,omitempty"`
	MaxWorkers int    `yaml:"max_workers"`
}

type CacheConfig struct {
	Enabled bool `yaml:"enabled"`
	TTL     int  `yaml:"ttl"`
}

type FormatConfig struct {
	Template       string `yaml:"template"`
	DateFormat     string `yaml:"date_format"`
	ServicePattern string `yaml:"service_pattern,omitempty"` // サービス名パターン（中間部分のみ）
}

func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			MaxWorkers: 3,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     0,
		},
		Format: FormatConfig{
			Template:       "{{.Date}}-{{.Service}}-{{.OriginalName}}",
			DateFormat:     "20060102",
			ServicePattern: "",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		if err := cfg.loadFromFile(path); err != nil {
			return nil, err
		}
	} else {
		defaultPath := DefaultConfigPath()
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			// 設定ファイルが存在しない場合は作成
			if err := createDefaultConfigFile(defaultPath); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
		}
		if err := cfg.loadFromFile(defaultPath); err != nil {
			return nil, err
		}
	}

	if err := cfg.resolveEnvVars(); err != nil {
		return nil, err
	}

	if err := cfg.autoDetectProvider(); err != nil {
		return nil, err
	}

	if err := cfg.setDefaultModel(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func createDefaultConfigFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	content := `# receipt-pdf-renamer configuration

# AI API settings
ai:
  # Provider: "anthropic" or "openai"
  # If not specified, auto-detected from environment variables
  # provider: "anthropic"

  # API key (can use environment variable reference)
  # If not specified, uses ANTHROPIC_API_KEY or OPENAI_API_KEY from environment
  # api_key: "${ANTHROPIC_API_KEY}"

  # Model name (optional, uses default if not specified)
  # Anthropic: claude-sonnet-4-20250514
  # OpenAI: gpt-4o
  # model: "claude-sonnet-4-20250514"

  # For OpenAI-compatible APIs (e.g., Ollama, LM Studio)
  # base_url: "http://localhost:11434/v1"

  # Number of parallel workers for analysis
  max_workers: 3

# Cache settings
cache:
  enabled: true
  ttl: 0  # Days until cache expires (0 = never expires)

# Rename format settings
format:
  # Output: YYYYMMDD-{service_pattern}-original.pdf
  # Available: {{.Service}} (service name from receipt)
  # Set your pattern before renaming (e.g., "{{.Service}}" or "MyCompany")
  service_pattern: ""
  date_format: "20060102"  # Go date format (YYYYMMDD)
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func (c *Config) resolveEnvVars() error {
	c.AI.APIKey = expandEnvVar(c.AI.APIKey)
	c.AI.BaseURL = expandEnvVar(c.AI.BaseURL)
	return nil
}

func expandEnvVar(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envName := s[2 : len(s)-1]
		return os.Getenv(envName)
	}
	return s
}

func (c *Config) autoDetectProvider() error {
	if c.AI.Provider != "" && c.AI.APIKey != "" {
		return nil
	}

	if c.AI.APIKey == "" {
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			c.AI.Provider = "anthropic"
			c.AI.APIKey = key
			return nil
		}

		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			c.AI.Provider = "openai"
			c.AI.APIKey = key
			return nil
		}

		return fmt.Errorf("no API key found: set ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable, or specify in config file")
	}

	if c.AI.Provider == "" {
		c.AI.Provider = "anthropic"
	}

	return nil
}

func (c *Config) setDefaultModel() error {
	if c.AI.Model != "" {
		return nil
	}

	switch c.AI.Provider {
	case "anthropic":
		c.AI.Model = "claude-sonnet-4-20250514"
	case "openai":
		c.AI.Model = "gpt-4o"
	default:
		return fmt.Errorf("unknown provider: %s", c.AI.Provider)
	}

	return nil
}

func (c *Config) ProviderDisplayName() string {
	switch c.AI.Provider {
	case "anthropic":
		return "Anthropic Claude API"
	case "openai":
		if c.AI.BaseURL != "" {
			return fmt.Sprintf("OpenAI-compatible API (%s)", c.AI.BaseURL)
		}
		return "OpenAI API"
	default:
		return c.AI.Provider
	}
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "receipt-pdf-renamer", "config.yaml")
}

func DefaultCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "receipt-pdf-renamer")
}

// SaveConfig は設定をグローバル設定ファイルに保存する
// Note: APIキーはKeyringで管理するため、ファイルには保存しない
func (c *Config) Save() error {
	path := DefaultConfigPath()

	// APIキーを除いた設定を保存用にコピー
	saveConfig := &Config{
		AI: AIConfig{
			Provider:   c.AI.Provider,
			BaseURL:    c.AI.BaseURL,
			Model:      c.AI.Model,
			MaxWorkers: c.AI.MaxWorkers,
			// APIKey は保存しない（Keyringで管理）
		},
		Cache: c.Cache,
		Format: FormatConfig{
			ServicePattern: c.Format.ServicePattern,
			DateFormat:     c.Format.DateFormat,
			// Template は ServicePattern から自動生成されるため保存不要
		},
	}

	data, err := yaml.Marshal(saveConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := `# receipt-pdf-renamer configuration
# This file is managed by the GUI application.
# API keys are stored securely in the system keyring.

`
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LocalConfigFileName はローカル設定ファイル名
const LocalConfigFileName = ".receipt-pdf-renamer.yaml"

// LoadWithLocal はグローバル設定を読み込み、ローカル設定で上書きする
func LoadWithLocal(globalPath, directory string) (*Config, error) {
	cfg, err := Load(globalPath)
	if err != nil {
		return nil, err
	}

	// ローカル設定ファイルを読み込んで上書き
	localPath := filepath.Join(directory, LocalConfigFileName)
	if _, err := os.Stat(localPath); err == nil {
		// ローカル設定を一時的に読み込み
		localCfg := &Config{}
		if err := localCfg.loadFromFile(localPath); err != nil {
			// ローカル設定の読み込みに失敗した場合は警告を出して続行
			fmt.Fprintf(os.Stderr, "Warning: failed to load local config %s: %v\n", localPath, err)
		} else {
			// サービスパターンが設定されている場合は検証して適用
			if localCfg.Format.ServicePattern != "" {
				fullTemplate := BuildFullTemplate(localCfg.Format.ServicePattern)
				if err := ValidateTemplate(fullTemplate); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: invalid service_pattern in %s: %v (using global config)\n", localPath, err)
				} else {
					cfg.Format.ServicePattern = localCfg.Format.ServicePattern
					cfg.Format.Template = fullTemplate
				}
			}
		}
	}

	return cfg, nil
}

// BuildFullTemplate はサービスパターンからフルテンプレートを構築する
func BuildFullTemplate(servicePattern string) string {
	return "{{.Date}}-" + servicePattern + "-{{.OriginalName}}"
}

// ValidateTemplate はテンプレートが有効かどうかを検証する
func ValidateTemplate(templateStr string) error {
	_, err := template.New("test").Parse(templateStr)
	return err
}

// LocalConfig はローカル設定ファイルに保存する内容（変更点のみ）
type LocalConfig struct {
	Format *LocalFormatConfig `yaml:"format,omitempty"`
}

type LocalFormatConfig struct {
	ServicePattern string `yaml:"service_pattern,omitempty"`
}

// SaveLocalConfig はローカル設定をカレントディレクトリに保存
func SaveLocalConfig(directory string, servicePattern string) error {
	localPath := filepath.Join(directory, LocalConfigFileName)

	// 既存のローカル設定を読み込む
	local := &LocalConfig{}
	if data, err := os.ReadFile(localPath); err == nil {
		_ = yaml.Unmarshal(data, local) // エラーは無視してデフォルト値で続行
	}

	// サービスパターンを更新
	if local.Format == nil {
		local.Format = &LocalFormatConfig{}
	}
	local.Format.ServicePattern = servicePattern

	// YAMLに変換
	data, err := yaml.Marshal(local)
	if err != nil {
		return fmt.Errorf("failed to marshal local config: %w", err)
	}

	// ファイルに書き込み
	content := "# Local overrides for receipt-pdf-renamer\n# This file overrides ~/.config/receipt-pdf-renamer/config.yaml\n\n" + string(data)
	if err := os.WriteFile(localPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write local config: %w", err)
	}

	return nil
}
