# CLAUDE.md

このファイルは Claude Code がこのリポジトリで作業する際のガイダンスを提供します。

## プロジェクト概要

receipt-pdf-renamer は、領収書PDFファイルをAIで解析し、支払日とサービス名を抽出して自動リネームするCLIツールです。

- **TUIモード**: 対話的にファイルを選択してリネーム
- **Headlessモード**: `--exec` フラグでバッチ処理

## ビルド・テストコマンド

```bash
make build      # ./bin/receipt-pdf-renamer にビルド
make test       # テスト実行
make run        # ビルドして実行
make clean      # ビルド成果物を削除
make fmt        # コードフォーマット
make lint       # Lint実行 (golangci-lint)
make tidy       # go mod tidy
```

## アーキテクチャ

```
cmd/receipt-pdf-renamer/    # エントリーポイント
internal/
  ai/                       # AI プロバイダー (Anthropic, OpenAI)
  cache/                    # 解析結果キャッシュ (SHA256ハッシュベース)
  config/                   # 設定管理 (グローバル + ローカル)
  pdf/                      # PDF→画像変換 (OpenAI用)
  renamer/                  # ファイルリネーム処理
  tui/                      # TUI (bubbletea)
```

## 設定ファイル

- **グローバル**: `~/.config/receipt-pdf-renamer/config.yaml`
- **ローカル**: `.receipt-pdf-renamer.yaml` (カレントディレクトリ、グローバルを上書き)
- **キャッシュ**: `~/.cache/receipt-pdf-renamer/`

## コーディング規約

- Go標準のフォーマット (`gofmt`)
- エラーは適切にラップして返す (`fmt.Errorf("context: %w", err)`)
- 日本語コメントOK
- TUIのスタイルは `internal/tui/view.go` の lipgloss スタイル定義に従う

## 重要な型

- `ai.ReceiptInfo`: AI解析結果 (Date, Service)
- `tui.FileItem`: ファイル状態管理
- `tui.ConfigInfo`: TUI表示用設定情報
- `config.Config`: アプリケーション設定

## リネームフォーマット

固定形式: `YYYYMMDD-{ServicePattern}-{OriginalName}.pdf`

- `ServicePattern` はユーザーが編集可能 (デフォルト: `{{.Service}}`)
- 日付部分とオリジナルファイル名部分は固定

## テスト時の注意

- AI APIキーが必要 (`ANTHROPIC_API_KEY` または `OPENAI_API_KEY`)
- PDFファイルは `.gitignore` で除外されている
