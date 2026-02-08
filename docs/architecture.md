# プロジェクト構造・アーキテクチャ

## ディレクトリ構造

```
receipt-pdf-renamer/
├── main.go                    # Wailsエントリーポイント
├── app.go                     # Appコア（バックエンドAPI）
├── internal/
│   ├── ai/
│   │   ├── provider.go        # Provider インターフェース
│   │   ├── anthropic.go       # Anthropic Claude 実装
│   │   └── openai.go          # OpenAI/互換 実装
│   ├── config/
│   │   └── config.go          # 設定ファイル読み込み・保存
│   ├── pdf/
│   │   └── converter.go       # PDF → 画像変換（OpenAI用）
│   ├── cache/
│   │   └── cache.go           # キャッシュ管理
│   └── renamer/
│       └── renamer.go         # リネームロジック
├── frontend/                  # Svelteフロントエンド
│   ├── src/
│   │   ├── App.svelte         # メイン画面
│   │   ├── lib/
│   │   │   └── Settings.svelte # 設定画面
│   │   └── main.ts
│   ├── package.json
│   └── pnpm-lock.yaml
├── build/
│   ├── darwin/
│   │   └── Info.plist         # macOS設定（ファイル関連付け含む）
│   └── windows/
│       ├── context-menu-install.reg
│       └── context-menu-uninstall.reg
├── .github/workflows/
│   ├── ci.yml                 # CI（build, test, lint, fmt, tidy）
│   ├── build.yml              # ビルド確認（macOS/Windows）
│   └── release.yml            # リリース（tag push時）
├── docs/
│   ├── architecture.md        # このファイル
│   ├── design-details.md      # 詳細設計
│   └── requirements.md        # 要件定義
├── go.mod
├── go.sum
├── wails.json
└── Makefile
```

---

## 技術スタック

| 項目 | 技術 |
|------|------|
| GUIフレームワーク | Wails v2 |
| フロントエンド | Svelte + TypeScript |
| バックエンド | Go |
| APIキー管理 | go-keyring（OS標準キーチェーン） |

---

## バックエンドAPI（app.go）

フロントエンドから呼び出されるメソッド:

### ファイル操作

| メソッド | 説明 |
|---------|------|
| `AddFiles(paths []string)` | PDFファイルを追加 |
| `GetFiles()` | ファイル一覧取得 |
| `ClearFiles()` | ファイル一覧クリア |
| `ToggleFileSelection(id int)` | 選択切り替え |
| `SelectAll()` / `DeselectAll()` | 全選択/全解除 |

### 解析・リネーム

| メソッド | 説明 |
|---------|------|
| `AnalyzeFiles()` | AI解析を開始（非同期） |
| `RenameFiles()` | 選択ファイルをリネーム |

### ダイアログ

| メソッド | 説明 |
|---------|------|
| `OpenFileDialog()` | ファイル選択ダイアログ |
| `OpenFolderDialog()` | フォルダ選択ダイアログ |
| `ScanFolder(path)` | フォルダ内のPDFをスキャン |

### 設定

| メソッド | 説明 |
|---------|------|
| `GetSettings()` | 現在の設定取得 |
| `SaveSettingsWithEndpoint(...)` | 設定保存 |
| `SaveAPIKey(provider, key)` | APIキーをキーチェーンに保存 |
| `GetAPIKey(provider)` | キーチェーンからAPIキー取得 |
| `DeleteAPIKey(provider)` | APIキー削除 |

### キャッシュ

| メソッド | 説明 |
|---------|------|
| `ClearCache()` | キャッシュクリア |
| `GetCacheCount()` | キャッシュ件数取得 |

---

## イベント（Backend → Frontend）

| イベント名 | タイミング |
|-----------|-----------|
| `files-updated` | ファイル状態が更新された時 |
| `analysis-complete` | 全ファイルの解析完了時 |

---

## AI プロバイダー抽象化

```go
// internal/ai/provider.go
type Provider interface {
    AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error)
}

type ReceiptInfo struct {
    Date    string // YYYYMMDD形式
    Service string // サービス名
}
```

### Anthropic 実装
- PDFを直接Base64エンコードして送信
- `application/pdf` として処理

### OpenAI/互換 実装
- PDFを画像に変換して送信（Vision API）
- `poppler` の `pdftoppm` が必要

---

## 処理フロー

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   main.go   │────▶│    App      │────▶│  Frontend   │
│  (Wails)    │     │  (app.go)   │     │  (Svelte)   │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │    cache    │ │ ai/provider │ │   renamer   │
    └─────────────┘ └──────┬──────┘ └─────────────┘
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

## ファイル状態（ItemStatus）

| 状態 | 説明 |
|------|------|
| `pending` | 追加直後、解析待ち |
| `analyzing` | AI解析中 |
| `ready` | 解析完了、リネーム可能 |
| `cached` | キャッシュから取得 |
| `renamed` | リネーム完了 |
| `error` | エラー発生 |
| `skipped` | スキップ（既にリネーム済み形式） |

---

## 依存ライブラリ

| パッケージ | 用途 |
|-----------|------|
| `github.com/wailsapp/wails/v2` | GUIフレームワーク |
| `github.com/anthropics/anthropic-sdk-go` | Anthropic Claude API |
| `github.com/sashabaranov/go-openai` | OpenAI API（互換含む） |
| `github.com/zalando/go-keyring` | OSキーチェーン連携 |
| `gopkg.in/yaml.v3` | 設定ファイル |
