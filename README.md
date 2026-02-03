# receipt-pdf-renamer

領収書PDFファイルをAIで解析し、支払日とサービス名を抽出して自動リネームするCLIツール。

## 特徴

- **AI解析**: Anthropic Claude または OpenAI GPT でPDFから情報を抽出
- **TUIモード**: 対話的にファイルを選択・プレビュー・リネーム
- **Headlessモード**: CI/スクリプトでのバッチ処理に対応
- **キャッシュ**: 解析結果をキャッシュして再実行を高速化
- **ローカル設定**: ディレクトリごとにフォーマットをカスタマイズ可能

## インストール

```bash
go install github.com/naotama2002/receipt-pdf-renamer/cmd/receipt-pdf-renamer@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/naotama2002/receipt-pdf-renamer.git
cd receipt-pdf-renamer
make build
```

## 必要条件

- Go 1.21+
- APIキー (いずれか):
  - `ANTHROPIC_API_KEY` - Anthropic Claude API
  - `OPENAI_API_KEY` - OpenAI API

## 使い方

### TUIモード (デフォルト)

```bash
# カレントディレクトリのPDFを処理
receipt-pdf-renamer

# 指定ディレクトリのPDFを処理
receipt-pdf-renamer /path/to/receipts
```

**キー操作:**
- `↑/↓` - カーソル移動
- `Space` - 選択/解除
- `a` - 全選択
- `n` - 全解除
- `t` - フォーマット編集
- `Enter` - リネーム実行
- `q` - 終了

### Headlessモード

```bash
# バッチ処理
receipt-pdf-renamer --exec --path=/path/to/receipts

# ドライラン (実際にはリネームしない)
receipt-pdf-renamer --exec --dry-run
```

### オプション

```
--exec          Headlessモード (TUIなし)
--path          対象ディレクトリ (--exec時)
--dry-run       プレビューのみ (リネームしない)
--config        設定ファイルパス
--workers       並列ワーカー数
--no-cache      キャッシュを使用しない
--clear-cache   キャッシュをクリア
```

## 出力フォーマット

```
YYYYMMDD-{ServicePattern}-{OriginalName}.pdf
```

例: `20250101-Amazon-receipt-001.pdf`

- **YYYYMMDD**: 支払日 (AIが抽出)
- **ServicePattern**: サービス名パターン (編集可能、デフォルト: `{{.Service}}`)
- **OriginalName**: 元のファイル名

## 設定

### グローバル設定

`~/.config/receipt-pdf-renamer/config.yaml` (初回起動時に自動作成)

```yaml
ai:
  # provider: "anthropic"  # または "openai"
  # model: "claude-sonnet-4-20250514"
  max_workers: 3

cache:
  enabled: true
  ttl: 0  # 0 = 無期限

format:
  service_pattern: "{{.Service}}"
  date_format: "20060102"
```

### ローカル設定

カレントディレクトリに `.receipt-pdf-renamer.yaml` を作成すると、グローバル設定を上書きできます:

```yaml
format:
  service_pattern: "{{.Service}}-経費"
```

TUIで `t` キーを押してフォーマットを編集すると、このファイルが自動作成されます。

## 対応AIプロバイダー

| プロバイダー | 環境変数 | デフォルトモデル |
|------------|---------|----------------|
| Anthropic | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 |
| OpenAI | `OPENAI_API_KEY` | gpt-4o |

OpenAI互換API (Ollama, LM Studio等) を使用する場合は、設定ファイルで `base_url` を指定してください。

## ライセンス

MIT
