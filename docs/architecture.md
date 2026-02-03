# プロジェクト構造・アーキテクチャ

## ディレクトリ構造

```
receipt-pdf-renamer/
├── cmd/
│   └── receipt-pdf-renamer/
│       └── main.go              # エントリポイント
├── internal/
│   ├── ai/
│   │   ├── provider.go          # Provider インターフェース
│   │   ├── anthropic.go         # Anthropic Claude 実装
│   │   └── openai.go            # OpenAI/互換 実装
│   ├── config/
│   │   └── config.go            # 設定ファイル読み込み
│   ├── pdf/
│   │   └── converter.go         # PDF → 画像変換（OpenAI用）
│   ├── cache/
│   │   └── cache.go             # キャッシュ管理
│   ├── renamer/
│   │   └── renamer.go           # リネームロジック
│   └── tui/
│       ├── model.go             # bubbletea Model
│       ├── view.go              # 画面描画
│       └── update.go            # イベント処理
├── docs/
│   ├── requirements.md          # 要件定義
│   ├── architecture.md          # このファイル
│   └── design-details.md        # 詳細設計
├── go.mod
├── go.sum
└── Makefile
```

---

## AI プロバイダー抽象化

### Provider インターフェース

```go
// internal/ai/provider.go
type Provider interface {
    AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error)
}

type ReceiptInfo struct {
    Date    string // YYYYMMDD形式
    Service string // サービス名
}

// ファクトリー関数
func NewProvider(cfg *config.AIConfig) (Provider, error) {
    switch cfg.Provider {
    case "anthropic":
        return NewAnthropicProvider(cfg)
    case "openai":
        return NewOpenAIProvider(cfg)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

### Anthropic 実装

```go
// internal/ai/anthropic.go
type AnthropicProvider struct {
    client *anthropic.Client
    model  string
}

func (p *AnthropicProvider) AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error) {
    // PDFを直接Base64エンコードして送信
    pdfData, _ := os.ReadFile(pdfPath)
    base64PDF := base64.StdEncoding.EncodeToString(pdfData)

    // Claude APIに送信（application/pdf）
    // ...
}
```

### OpenAI/互換 実装

```go
// internal/ai/openai.go
type OpenAIProvider struct {
    client    *openai.Client
    model     string
    converter *pdf.Converter
}

func (p *OpenAIProvider) AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error) {
    // PDFを画像に変換
    imageData, _ := p.converter.ToImage(pdfPath)
    base64Image := base64.StdEncoding.EncodeToString(imageData)

    // Vision APIに送信
    // ...
}
```

---

## PDF → 画像変換（OpenAI用）

OpenAI/互換APIはPDFを直接受け付けないため、画像に変換が必要。

```go
// internal/pdf/converter.go
type Converter struct{}

func (c *Converter) ToImage(pdfPath string) ([]byte, error) {
    // poppler の pdftoppm を使用
    cmd := exec.Command("pdftoppm", "-png", "-singlefile", pdfPath, "-")
    return cmd.Output()
}

func (c *Converter) IsAvailable() bool {
    _, err := exec.LookPath("pdftoppm")
    return err == nil
}
```

**注意**: OpenAI互換を使う場合は `poppler` のインストールが必要

```bash
# macOS
brew install poppler

# Ubuntu
apt install poppler-utils
```

---

## 設定

```go
// internal/config/config.go
type Config struct {
    AI     AIConfig     `yaml:"ai"`
    Cache  CacheConfig  `yaml:"cache"`
    Format FormatConfig `yaml:"format"`
}

type AIConfig struct {
    Provider   string `yaml:"provider"`    // "anthropic" or "openai"
    BaseURL    string `yaml:"base_url"`    // OpenAI互換用（オプション）
    APIKey     string `yaml:"api_key"`
    Model      string `yaml:"model"`
    MaxWorkers int    `yaml:"max_workers"`
}

type CacheConfig struct {
    Enabled bool `yaml:"enabled"`
    TTL     int  `yaml:"ttl"`  // 日数、0=無期限
}

type FormatConfig struct {
    Template   string `yaml:"template"`
    DateFormat string `yaml:"date_format"`
}
```

---

## 依存ライブラリ

| パッケージ | 用途 |
|-----------|------|
| `github.com/charmbracelet/bubbletea` | TUIフレームワーク |
| `github.com/charmbracelet/lipgloss` | TUIスタイリング |
| `github.com/anthropics/anthropic-sdk-go` | Anthropic Claude API |
| `github.com/sashabaranov/go-openai` | OpenAI API（互換含む） |
| `gopkg.in/yaml.v3` | 設定ファイル |
| `github.com/spf13/cobra` | CLIフラグ |

---

## 処理フロー

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   main.go   │────▶│   config    │────▶│    tui      │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                           ┌───────────────────┼───────────────────┐
                           │                   │                   │
                           ▼                   ▼                   ▼
                    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
                    │    cache    │     │ ai/provider │     │   renamer   │
                    └─────────────┘     └──────┬──────┘     └─────────────┘
                                               │
                                   ┌───────────┴───────────┐
                                   │                       │
                                   ▼                       ▼
                            ┌─────────────┐         ┌─────────────┐
                            │  anthropic  │         │   openai    │
                            │ (PDF直接)   │         │ (画像変換)  │
                            └─────────────┘         └──────┬──────┘
                                                           │
                                                           ▼
                                                    ┌─────────────┐
                                                    │ pdf/convert │
                                                    └─────────────┘
```

---

## 実行モード

### モード分岐

```
┌─────────────┐
│   main.go   │
└──────┬──────┘
       │
       ▼
   --exec ?
       │
   ┌───┴───┐
   │       │
   ▼       ▼
  Yes      No
   │       │
   ▼       ▼
┌──────┐  ┌──────┐
│ヘッド│  │ TUI  │
│レス  │  │モード│
└──────┘  └──────┘
```

### TUIモード 状態遷移

```
                    ┌─────────────┐
                    │   起動      │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ PDFスキャン │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  AI解析中   │ ← 並列処理（max_workers制御）
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
        ┌──────────│ ファイル選択 │──────────┐
        │          └──────┬──────┘          │
        │                 │                  │
        ▼                 ▼                  ▼
   [Space]           [Enter]              [q]
   選択切替        リネーム実行            終了
        │                 │
        │                 ▼
        │          ┌─────────────┐
        │          │  完了表示   │
        │          └─────────────┘
        │                 │
        └─────────────────┘
```

### ヘッドレスモード フロー

```
┌─────────────┐
│   起動      │
│ (--exec)    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ PDFスキャン │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  AI解析     │ ← 並列処理
│ (進捗表示)  │
└──────┬──────┘
       │
       ▼
   --dry-run ?
       │
   ┌───┴───┐
   │       │
   ▼       ▼
  Yes      No
   │       │
   ▼       ▼
┌──────┐  ┌──────┐
│結果  │  │リネーム│
│表示  │  │実行   │
└──────┘  └──────┘
       │
       ▼
┌─────────────┐
│ 完了サマリ  │
│ (exit code) │
└─────────────┘
```

### Exit Code（ヘッドレスモード）

| Code | 意味 |
|------|------|
| 0 | 全て成功 |
| 1 | 一部失敗 |
| 2 | 全て失敗 / エラー |
