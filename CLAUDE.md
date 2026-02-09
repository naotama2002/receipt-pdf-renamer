# CLAUDE.md

このファイルは Claude Code がこのリポジトリで作業する際のガイダンスを提供します。

## プロジェクト概要

receipt-pdf-renamer は、領収書PDFファイルをAIで解析し、支払日とサービス名を抽出して自動リネームするGUIアプリです。

- **Wails v2** + **Svelte** で構築
- **macOS / Windows** 対応
- ドラッグ&ドロップでファイル追加
- 「このアプリで開く」対応（macOS: OnFileOpen, Windows: コマンドライン引数）
- Windowsコンテキストメニュー登録用 .reg ファイル付属

## ビルド・開発コマンド

```bash
make dev              # 開発モード（ホットリロード）
make build            # 現在のプラットフォーム用ビルド
make build-mac        # macOS用ビルド（Universal Binary）
make build-win        # Windows用ビルド
make test             # テスト実行
make clean            # ビルド成果物を削除
make fmt              # コードフォーマット
make lint             # Lint実行 (golangci-lint)
make tidy             # go mod tidy
make install-frontend # フロントエンド依存インストール
make generate         # Wailsバインディング更新
```

## アーキテクチャ

```
main.go                 # Wailsエントリーポイント
app.go                  # Wails App構造体・バックエンドAPI
internal/
  ai/                   # AI プロバイダー (Anthropic Claude)
  cache/                # 解析結果キャッシュ (SHA256ハッシュベース)
  config/               # 設定管理
  renamer/              # ファイルリネーム処理
frontend/
  src/
    App.svelte          # メインコンポーネント
    lib/
      Settings.svelte   # 設定モーダル
  wailsjs/              # 自動生成バインディング
build/
  darwin/
    Info.plist          # macOS設定（PDF関連付け含む）
  windows/
    context-menu-install.reg   # 右クリックメニュー登録
    context-menu-uninstall.reg # 右クリックメニュー削除
```

## 設定ファイル

- **グローバル設定**: `~/.config/receipt-pdf-renamer/config.yaml`
- **キャッシュ**: `~/.cache/receipt-pdf-renamer/`
- **APIキー**: OSセキュアストレージ（`go-keyring`経由）
  - macOS: Keychain
  - Windows: Credential Manager
  - サービス名: `receipt-pdf-renamer`
  - キー名: `{provider}-api-key` (例: `anthropic-api-key`)

## コーディング規約

- Go標準のフォーマット (`gofmt`)
- エラーは適切にラップして返す (`fmt.Errorf("context: %w", err)`)
- 日本語コメントOK
- フロントエンドはSvelte + TypeScript

## 重要な型

- `ai.ReceiptInfo`: AI解析結果 (Date, Service)
- `FileItem`: ファイル状態管理（app.go内）
- `ConfigInfo`: 設定情報DTO
- `SettingsInfo`: 設定画面用DTO
- `config.Config`: アプリケーション設定

## 主なバックエンドAPI（app.go）

- `AddFiles(paths)` - ファイル追加
- `AnalyzeFiles()` - AI解析実行
- `RenameFiles()` - リネーム実行
- `GetSettings()` / `SaveSettingsWithModel()` - 設定取得・保存
- `SaveAPIKey()` / `GetAPIKey()` / `DeleteAPIKey()` - APIキー管理（Keyring）
- `GetAvailableModels()` - 利用可能モデル取得

## リネームフォーマット

固定形式: `YYYYMMDD-{ServicePattern}-{OriginalName}.pdf`

- `ServicePattern` はユーザーがGUI上で編集可能 (デフォルト: `{{.Service}}`)
- 日付部分とオリジナルファイル名部分は固定

## テスト時の注意

- AI APIキーが必要 (`ANTHROPIC_API_KEY`)
- PDFファイルは `.gitignore` で除外されている
- `wails dev` で開発モード起動

## パッケージマネージャ

- フロントエンド: **pnpm**
- バックエンド: Go modules
