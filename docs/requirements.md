# receipt-pdf-renamer 要件定義

## 概要

領収書PDFファイルを自動的にリネームするTUIツール。
Anthropic Claude APIを使ってPDFから支払日とサービス名を抽出し、指定形式にリネームする。

## 決定事項

| 項目 | 決定 |
|------|------|
| 開発言語 | Go |
| TUIフレームワーク | bubbletea |
| AI API | Anthropic Claude API / OpenAI互換API（ローカルLLM対応） |
| 対応サービス | 複数（Cursor, GitHub, AWS など）- AI自動判定 |
| ファイル名形式 | テンプレートで設定可能 |
| TUI編集機能 | なし（シンプルに） |
| 実行モード | TUIモード / ヘッドレスモード（--exec） |

---

## 機能要件

### 基本機能

1. **PDFスキャン**
   - 指定ディレクトリ（デフォルト: カレント）内のPDFを一覧表示
   - サブディレクトリは対象外

2. **AI解析**
   - PDFを直接Claude APIに送信（画像変換不要）
   - 抽出情報:
     - 支払日（Paid date）
     - サービス名（Cursor, GitHub, AWS など）

3. **リネームプレビュー**
   - 変更前 → 変更後を一覧表示
   - 例: `Receipt-2009-4865.pdf` → `20250115-Cursor-Receipt-2009-4865.pdf`

4. **リネーム実行**
   - 選択したファイルのみ、または一括でリネーム

### TUI画面構成

```
┌─ receipt-pdf-renamer ─────────────────────────────────────────┐
│                                                                │
│  📁 /Users/naotama/Downloads                                   │
│                                                                │
│  [x] Receipt-2009-4865.pdf                                     │
│      → 20250115-Cursor-Receipt-2009-4865.pdf                   │
│                                                                │
│  [x] Invoice-12345.pdf                                         │
│      → 20250120-GitHub-Invoice-12345.pdf                       │
│                                                                │
│  [ ] Receipt-AWS-2025.pdf                                      │
│      → 20250101-AWS-Receipt-AWS-2025.pdf                       │
│                                                                │
│  ──────────────────────────────────────────────────────────── │
│  [Enter] リネーム実行  [Space] 選択切替  [a] 全選択  [q] 終了  │
└────────────────────────────────────────────────────────────────┘
```

### CLI インターフェース

```bash
# TUIモード（デフォルト）
receipt-pdf-renamer [directory]

# ヘッドレスモード（UIなし、自動実行）
receipt-pdf-renamer --exec [directory]
receipt-pdf-renamer --exec --path=/path/to/receipts
receipt-pdf-renamer --exec --path=/path/to/receipts --config=~/.config/receipt-pdf-renamer/work.yaml

# オプション
  --exec               UIなしで実行（ヘッドレスモード）
  --path string        対象ディレクトリ（--exec時に使用）
  --config string      設定ファイルパス
  --dry-run            リネームを実行せずプレビューのみ
  --clear-cache        キャッシュをクリア
  --no-cache           キャッシュを使用しない（今回のみ）
  --workers int        並列処理数を指定
  -h, --help           ヘルプ
```

### 実行モード

| モード | 説明 | 用途 |
|--------|------|------|
| **TUIモード** | インタラクティブUI | 手動でファイル選択・確認 |
| **ヘッドレスモード** | UIなし自動実行 | スクリプト、自動化、CI/CD |

#### ヘッドレスモードの動作

```bash
# 全PDFを自動リネーム
receipt-pdf-renamer --exec --path=/path/to/receipts

# 出力例
Scanning /path/to/receipts...
Found 3 PDF files

[1/3] Receipt-001.pdf
      → 20250115-Cursor-Receipt-001.pdf ✓

[2/3] Receipt-002.pdf
      → 20250120-GitHub-Receipt-002.pdf ✓

[3/3] Receipt-003.pdf
      ✗ Error: Failed to analyze (API error)

Completed: 2 renamed, 1 failed
```

#### ドライラン + ヘッドレス

```bash
# 実行せずにプレビューだけ（スクリプトでの確認用）
receipt-pdf-renamer --exec --dry-run --path=/path/to/receipts
```

---

## ファイル配置

| 種類 | パス | 説明 |
|------|------|------|
| 設定ファイル | `~/.config/receipt-pdf-renamer/config.yaml` | XDG Base Directory準拠 |
| キャッシュ | `~/.cache/receipt-pdf-renamer/` | システムのキャッシュクリアで削除可 |

### 環境変数による自動検出

設定ファイルがない場合、環境変数から自動でプロバイダーを検出します。

**優先順位:**
1. `ANTHROPIC_API_KEY` があれば → Anthropic Claude を使用
2. `OPENAI_API_KEY` があれば → OpenAI を使用
3. どちらもなければ → エラー

```bash
# 設定ファイルなしで実行
export ANTHROPIC_API_KEY=sk-ant-xxx
receipt-pdf-renamer
# → "Using Anthropic Claude API (auto-detected from ANTHROPIC_API_KEY)"

export OPENAI_API_KEY=sk-xxx
receipt-pdf-renamer
# → "Using OpenAI API (auto-detected from OPENAI_API_KEY)"
```

---

## 設定ファイル（オプション）

`~/.config/receipt-pdf-renamer/config.yaml`

設定ファイルがある場合は、そちらの設定が優先されます。

### Anthropic Claude API の場合

```yaml
ai:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-20250514"
  max_workers: 3

cache:
  enabled: true
  ttl: 0

format:
  template: "{{.Date}}-{{.Service}}-{{.OriginalName}}"
  date_format: "20060102"
```

### OpenAI API の場合

```yaml
ai:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o"
  max_workers: 3

cache:
  enabled: true
  ttl: 0

format:
  template: "{{.Date}}-{{.Service}}-{{.OriginalName}}"
  date_format: "20060102"
```

### ローカルLLM（OpenAI互換）の場合

```yaml
ai:
  provider: "openai"
  base_url: "http://localhost:11434/v1"  # Ollama の例
  api_key: "ollama"                       # ダミー値でOK
  model: "llama3.2-vision"
  max_workers: 1                          # ローカルは並列少なめ

cache:
  enabled: true
  ttl: 0

format:
  template: "{{.Date}}-{{.Service}}-{{.OriginalName}}"
  date_format: "20060102"
```

### 設定項目

| 項目 | 説明 |
|------|------|
| `ai.provider` | `anthropic` または `openai` |
| `ai.base_url` | APIエンドポイント（OpenAI互換のみ、省略時はOpenAI公式） |
| `ai.api_key` | APIキー（環境変数参照可） |
| `ai.model` | モデル名 |
| `ai.max_workers` | 並列処理数 |
| `cache.enabled` | キャッシュ有効/無効 |
| `cache.ttl` | キャッシュ有効期限（日数、0=無期限） |
| `format.template` | ファイル名テンプレート |
| `format.date_format` | 日付フォーマット（Go形式） |

### デフォルト値（設定ファイルなしの場合）

| 項目 | デフォルト値 |
|------|-------------|
| `ai.provider` | 環境変数から自動検出 |
| `ai.model` | Anthropic: `claude-sonnet-4-20250514` / OpenAI: `gpt-4o` |
| `ai.max_workers` | `3` |
| `cache.enabled` | `true` |
| `cache.ttl` | `0`（無期限） |
| `format.template` | `{{.Date}}-{{.Service}}-{{.OriginalName}}` |
| `format.date_format` | `20060102`（YYYYMMDD） |

---

## 技術詳細

### AI API 連携

#### 対応プロバイダー

| プロバイダー | 説明 |
|-------------|------|
| **Anthropic** | Claude API（PDF直接送信対応） |
| **OpenAI** | GPT-4o など（画像として送信） |
| **OpenAI互換** | Ollama, LM Studio, vLLM など |

#### PDF送信方法

- **Anthropic**: PDFを直接Base64エンコードして送信（`application/pdf`）
- **OpenAI/互換**: PDFを画像に変換して送信（Vision API）

※ OpenAI互換の場合のみ、PDF→画像変換が必要（poppler使用）

#### プロンプト例

```
この領収書から以下の情報を抽出してください：
1. 支払日（Paid date）をYYYYMMDD形式で
2. サービス/会社名

JSON形式で回答：
{"date": "20250115", "service": "Cursor"}
```

### 処理フロー

```
1. ディレクトリスキャン（*.pdf）
2. 各PDFのハッシュを計算
3. キャッシュチェック
4. Claude APIで解析（並列処理、max_workers制御）
5. 解析結果をキャッシュに保存
6. TUIで結果表示
7. ユーザー選択
8. リネーム実行
```

---

## 次のステップ

1. [x] 要件定義
2. [ ] プロジェクト構造設計
3. [ ] 実装開始
