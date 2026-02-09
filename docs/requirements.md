# 要件定義

## 概要

領収書PDFファイルを自動的にリネームするGUIアプリケーション。
AI APIを使ってPDFから支払日とサービス名を抽出し、指定形式にリネームする。

## 対応プラットフォーム

| OS | 対応 |
|----|------|
| macOS | Universal Binary (Intel/Apple Silicon) |
| Windows | amd64 |

---

## 機能要件

### 基本機能

1. **ファイル追加**
   - ドラッグ&ドロップでPDFを追加
   - ファイル選択ダイアログ
   - フォルダ選択→内部のPDFをスキャン

2. **AI解析**
   - PDFからAI APIで情報を抽出
   - 抽出情報: 支払日（YYYYMMDD）、サービス名
   - 並列処理対応（設定可能）

3. **リネームプレビュー**
   - 変更前 → 変更後を一覧表示
   - 例: `Receipt-001.pdf` → `20250115-Cursor-Receipt-001.pdf`

4. **リネーム実行**
   - 選択したファイルをリネーム

5. **OS連携**
   - macOS: Finderの「このアプリケーションで開く」対応
   - Windows: 右クリックコンテキストメニュー対応（レジストリ登録）

---

## AIプロバイダー

| プロバイダー | 説明 |
|-------------|------|
| **Anthropic** | Claude API（PDF直接送信対応） |

### PDF送信方法

- PDFを直接Base64エンコードして送信

---

## 設定

### 設定項目

| 項目 | 説明 |
|------|------|
| `ai.model` | モデル名 |
| `ai.max_workers` | 並列処理数（デフォルト: 3） |
| `cache.enabled` | キャッシュ有効/無効 |
| `cache.ttl` | キャッシュ有効期限（日数、0=無期限） |
| `format.service_pattern` | サービス部分のテンプレート |

### APIキー

- OS標準のキーチェーンに保存（設定ファイルには保存しない）
- macOS: Keychain
- Windows: Credential Manager

### ファイル配置

| 種類 | パス |
|------|------|
| 設定ファイル | `~/.config/receipt-pdf-renamer/config.yaml` |
| キャッシュ | `~/.cache/receipt-pdf-renamer/` |

---

## リネーム形式

固定形式: `YYYYMMDD-{ServicePattern}-{OriginalName}.pdf`

例:
- `Receipt-001.pdf` → `20250115-Cursor-Receipt-001.pdf`
- `Invoice-12345.pdf` → `20250120-GitHub-Invoice-12345.pdf`

---

## 開発環境

### 必要なツール

| ツール | バージョン | 用途 |
|--------|----------|------|
| Go | go.modに記載 | バックエンド |
| Node.js | .tool-versionsに記載 | フロントエンド |
| pnpm | .tool-versionsに記載 | パッケージ管理 |
| Wails CLI | 最新 | GUIフレームワーク |

### セットアップ

```bash
# Wails CLIインストール
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# フロントエンド依存インストール
cd frontend && pnpm install

# 開発サーバー起動
make dev
```

### ビルド

```bash
make build          # 現在のプラットフォーム
make build-mac      # macOS
make build-win      # Windows
```

---

## CI/CD

### GitHub Actions

| ワークフロー | トリガー | 内容 |
|-------------|---------|------|
| `ci.yml` | push/PR to main | build, test, lint, fmt, tidy |
| `build.yml` | push/PR to main | macOS/Windowsビルド確認 |
| `release.yml` | tag `v*` | リリースビルド、GitHub Releases公開 |

### リリース成果物

| ファイル | 内容 |
|---------|------|
| `Receipt-PDF-Renamer-{version}-mac.dmg` | macOSインストーラー |
| `receipt-pdf-renamer-{version}-win.zip` | Windows実行ファイル + REGファイル |
