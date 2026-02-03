# 詳細設計 - 並列処理とキャッシュ

## 決定事項

| 項目 | 決定 |
|------|------|
| キャッシュ有効期限 | 設定可能（デフォルト: 無期限） |
| キャッシュクリア | `--clear-cache` フラグを用意 |
| 並列処理数 | 設定可能（デフォルト: 3） |

---

## 1. 並列処理

### 設計

```go
type ConcurrencyConfig struct {
    MaxWorkers int `yaml:"max_workers"` // 同時処理数（デフォルト: 3）
}
```

### 実装

```go
func (c *Client) AnalyzeMultiple(ctx context.Context, pdfPaths []string) []*Result {
    results := make([]*Result, len(pdfPaths))
    sem := make(chan struct{}, c.config.MaxWorkers)
    var wg sync.WaitGroup

    for i, path := range pdfPaths {
        wg.Add(1)
        go func(idx int, pdfPath string) {
            defer wg.Done()
            sem <- struct{}{}        // acquire
            defer func() { <-sem }() // release

            info, err := c.AnalyzeReceipt(ctx, pdfPath)
            results[idx] = &Result{Info: info, Error: err}
        }(i, path)
    }

    wg.Wait()
    return results
}
```

### TUIとの連携

各ファイルの解析状態をリアルタイム表示:

```
[x] Receipt-001.pdf  ✓ ready
    → 20250115-Cursor-Receipt-001.pdf

[ ] Receipt-002.pdf  ⏳ analyzing...

[ ] Receipt-003.pdf  ✗ error: API rate limit
```

---

## 2. キャッシュ機能

### キャッシュキー

**ファイルハッシュベース（SHA256）**
- ファイル内容のハッシュをキーに使用
- 同じ内容のPDFは場所が変わってもキャッシュヒット
- 内容が変わったら自動的に再解析

### キャッシュ保存場所

```
~/.cache/receipt-pdf-renamer/
└── analysis/
    ├── a1b2c3d4e5f6...json
    └── f6e5d4c3b2a1...json
```

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

### 有効期限の設定

```yaml
cache:
  enabled: true
  ttl: 0  # 0 = 無期限、それ以外は日数
```

### キャッシュクリアコマンド

```bash
# 全キャッシュをクリア
receipt-pdf-renamer --clear-cache

# 確認メッセージ付き
receipt-pdf-renamer --clear-cache
> キャッシュを削除しますか？ (y/N)
```

---

## 3. 設定ファイル（更新版）

`~/.config/receipt-pdf-renamer/config.yaml`

```yaml
# AI API設定
ai:
  api_key: "${ANTHROPIC_API_KEY}"  # 環境変数参照
  model: "claude-sonnet-4-20250514"
  max_workers: 3                    # 並列処理数

# キャッシュ設定
cache:
  enabled: true
  ttl: 0  # 有効期限（日数）、0 = 無期限

# リネーム形式
format:
  # 利用可能な変数: {{.Date}}, {{.Service}}, {{.OriginalName}}
  template: "{{.Date}}-{{.Service}}-{{.OriginalName}}"
  date_format: "20060102"  # Go形式（YYYYMMDD）
```

---

## 4. CLIオプション（更新版）

```bash
# 基本的な使い方
receipt-pdf-renamer [directory]

# オプション
  --config string    設定ファイルパス（デフォルト: ~/.config/receipt-pdf-renamer/config.yaml）
  --dry-run          リネームを実行せずプレビューのみ
  --clear-cache      キャッシュをクリア
  --no-cache         キャッシュを使用しない（今回のみ）
  --workers int      並列処理数（設定ファイルより優先）
  -h, --help         ヘルプ
```

---

## 5. 処理フロー（最終版）

```
1. CLI引数・設定ファイル読み込み
2. ディレクトリスキャン（*.pdf）
3. 各PDFのハッシュを計算
4. キャッシュチェック（enabled の場合）
   - ヒット（TTL内） → キャッシュから結果を取得
   - ミス or 期限切れ → 解析対象リストに追加
5. 解析対象を並列でClaude API解析（max_workers制御）
6. 解析結果をキャッシュに保存
7. TUIで結果表示
8. ユーザー選択
9. リネーム実行
```

---

## 6. TUI表示例

```
┌─ receipt-pdf-renamer ─────────────────────────────────────────┐
│                                                                │
│  📁 /Users/naotama/Downloads                                   │
│  📊 3 files found (1 cached, 2 analyzing)                      │
│                                                                │
│  [x] Receipt-001.pdf  ✓ ready (cached)                         │
│      → 20250115-Cursor-Receipt-001.pdf                         │
│                                                                │
│  [x] Receipt-002.pdf  ✓ ready                                  │
│      → 20250120-GitHub-Receipt-002.pdf                         │
│                                                                │
│  [ ] Receipt-003.pdf  ⏳ analyzing... (2/3)                     │
│                                                                │
│  ──────────────────────────────────────────────────────────── │
│  [Enter] リネーム実行  [Space] 選択切替  [a] 全選択  [q] 終了  │
└────────────────────────────────────────────────────────────────┘
```

---

## ディスカッションポイント

他に議論したいことはありますか？

1. **エラーハンドリング**: AI解析失敗時はスキップ？リトライ？
2. **ログ機能**: リネーム履歴を保存する？
3. **Undo機能**: リネームを元に戻す？
4. **それとも実装開始？**
