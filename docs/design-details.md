# 詳細設計

## 設定ファイル

`~/.config/receipt-pdf-renamer/config.yaml`

```yaml
ai:
  provider: "anthropic"       # "anthropic" or "openai"
  base_url: ""                # OpenAI互換用（オプション）
  model: "claude-sonnet-4-20250514"
  max_workers: 3

cache:
  enabled: true
  ttl: 0                      # 0 = 無期限

format:
  service_pattern: "{{.Service}}"
  date_format: "20060102"
```

**注意**: APIキーは設定ファイルではなく、OS標準のキーチェーンに保存される

---

## APIキー管理

`go-keyring` を使用してOSのセキュアストレージに保存:

| OS | 保存先 |
|----|--------|
| macOS | Keychain |
| Windows | Credential Manager |
| Linux | Secret Service (GNOME Keyring等) |

```go
// キー名: "receipt-pdf-renamer" / "{provider}-api-key"
keyring.Set("receipt-pdf-renamer", "anthropic-api-key", apiKey)
keyring.Get("receipt-pdf-renamer", "anthropic-api-key")
```

---

## キャッシュ

### 保存場所

```
~/.cache/receipt-pdf-renamer/
└── analysis/
    ├── a1b2c3d4e5f6...json
    └── f6e5d4c3b2a1...json
```

### キャッシュキー

ファイル内容のSHA256ハッシュを使用:
- 同じ内容のPDFは場所が変わってもキャッシュヒット
- 内容が変わったら自動的に再解析

### キャッシュファイル形式

```json
{
  "hash": "a1b2c3d4e5f6...",
  "analyzed_at": "2025-02-01T12:00:00Z",
  "result": {
    "date": "20250115",
    "service": "Cursor"
  }
}
```

---

## 並列処理

```go
maxWorkers := config.AI.MaxWorkers  // デフォルト: 3
sem := make(chan struct{}, maxWorkers)

for _, file := range files {
    go func(f FileItem) {
        sem <- struct{}{}        // acquire
        defer func() { <-sem }() // release
        analyzeFile(f)
    }(file)
}
```

---

## リネーム形式

固定形式: `YYYYMMDD-{ServicePattern}-{OriginalName}.pdf`

- `YYYYMMDD`: 支払日（AIが抽出）
- `ServicePattern`: ユーザー編集可能（デフォルト: `{{.Service}}`）
- `OriginalName`: 元のファイル名（拡張子除く）

### テンプレート変数

| 変数 | 説明 |
|------|------|
| `{{.Date}}` | 支払日（YYYYMMDD） |
| `{{.Service}}` | サービス名 |
| `{{.OriginalName}}` | 元ファイル名 |

---

## macOS「このアプリケーションで開く」対応

### Info.plist 設定

```xml
<key>CFBundleDocumentTypes</key>
<array>
  <dict>
    <key>CFBundleTypeExtensions</key>
    <array><string>pdf</string></array>
    <key>CFBundleTypeName</key>
    <string>PDF Document</string>
    <key>CFBundleTypeRole</key>
    <string>Viewer</string>
  </dict>
</array>
```

### 処理フロー

1. Finderで「このアプリケーションで開く」選択
2. macOSが `OnFileOpen(filePath)` を呼び出し
3. アプリがファイルを追加し、UIを更新

---

## Windows コンテキストメニュー対応

### レジストリ設定

`build/windows/context-menu-install.reg`:

```reg
[HKEY_CLASSES_ROOT\SystemFileAssociations\.pdf\shell\ReceiptPDFRenamer]
@="Receipt PDF Renamerで開く"

[HKEY_CLASSES_ROOT\SystemFileAssociations\.pdf\shell\ReceiptPDFRenamer\command]
@="\"C:\\path\\to\\receipt-pdf-renamer.exe\" \"%1\""
```

### 処理フロー

1. エクスプローラーで右クリック→「Receipt PDF Renamerで開く」
2. アプリがコマンドライン引数でファイルパスを受け取り
3. `DomReady()` でファイルを追加

---

## フロントエンド-バックエンド連携

### Wails バインディング

```typescript
// frontend/wailsjs/go/main/App.ts（自動生成）
export function AddFiles(paths: string[]): Promise<FileItem[]>;
export function AnalyzeFiles(): Promise<void>;
export function RenameFiles(): Promise<RenameResult>;
```

### イベント購読

```typescript
import { EventsOn } from '../wailsjs/runtime/runtime';

EventsOn('files-updated', (files: FileItem[]) => {
  // UIを更新
});
```

---

## ビルド

### 開発

```bash
make dev              # 開発サーバー起動
```

### プロダクション

```bash
make build            # 現在のプラットフォーム
make build-mac        # macOS Universal Binary
make build-win        # Windows amd64
make release-mac      # macOS（最適化、配布用）
make release-win      # Windows（最適化、配布用）
```

### 成果物

| プラットフォーム | 出力先 |
|----------------|--------|
| macOS | `build/bin/Receipt PDF Renamer.app` |
| Windows | `build/bin/receipt-pdf-renamer.exe` |
