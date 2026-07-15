---
name: git-commit
description: "Create commits from ALL local changes (unstaged + staged + untracked) at appropriate granularity on an appropriate branch. ONLY use when user explicitly invokes /git-commit slash command. Do NOT trigger on general git questions."
---

# Git Commit SKILL

このSKILLは **「git add 前のローカル変更（unstaged / staged / untracked）をすべて、適切な粒度で、適切なブランチにコミットする」** ために使う。
ユーザーが **`/git-commit` を明示的に呼んだときだけ** 実行する。

このプロジェクト（receipt-pdf-renamer）は **Wails v2 + Svelte** 製のGUIアプリ（領収書PDFを解析してリネームするツール）。
- バックエンド: `main.go`, `app.go`, `internal/ai`, `internal/cache`, `internal/config`, `internal/renamer`, `internal/history`, `internal/pdf`
- フロントエンド: `frontend/src/*.svelte`（Svelte + TypeScript）、生成バインディング `frontend/wailsjs/*`
- ビルド設定: `build/darwin`（Info.plist等）, `build/windows`（コンテキストメニュー .reg）

---

## General Rules
- コミットメッセージは **日本語** で書く
- **シークレット/鍵/トークンらしきものはコミットしない**（`.env*`, APIキー等。見つけたらコミットを止め、除去/無効化/ローテーションを提案）
- **生成物・テスト用PDFはコミットしない**（`build/bin/`, `frontend/node_modules/`, `frontend/dist/`, `*.pdf` は `.gitignore` 対象）
- 目標は「レビューしやすく、ロールバックしやすい」コミット:
  - 1コミット = 1目的
  - フォーマットのみの変更（`go fmt`）は機能変更と分ける
  - バックエンドAPI（`app.go` のメソッドシグネチャ）を変更した場合、`make generate` で再生成した `frontend/wailsjs/*` の差分は対応するバックエンド変更と同じコミットに含める

---

## Workflow

### 0) Preflight: 現状確認（必須）

```bash
git status --porcelain=v1
git rev-parse --abbrev-ref HEAD
```

- 変更が **0件** なら終了（「変更がありません」）
- 作業ブランチが `main` の場合は **必ず新規ブランチを作る**
  - `feat/short-description` / `fix/short-description` / `chore/short-description` 等

```bash
git checkout -b fix/short-description
```

---

### 1) すべての変更を把握する
```bash
git status
git diff --stat
git diff
git diff --cached
git ls-files --others --exclude-standard
```

---

### 2) コミット分割プランを作る（重要）

差分を見て、**先にコミットの分割単位を決める**:
- バックエンドの機能変更（`internal/ai`, `internal/cache`, `internal/config`, `internal/renamer`, `internal/history`, `internal/pdf`, `app.go`）
- フロントエンドの変更（`frontend/src/*.svelte`）
- Wailsバインディング再生成のみの差分（`frontend/wailsjs/*`）※バックエンドAPI変更に伴う場合は分けずに同梱
- フォーマットのみの変更（`go fmt`, Prettier等）
- 依存変更（`go.mod` / `go.sum`, `frontend/package.json` / `frontend/pnpm-lock.yaml`）
- CI / ビルド設定（`.github/workflows/`, `Makefile`, `build/darwin`, `build/windows`）
- ドキュメント（`README.md`, `CLAUDE.md`）

分割プランとして以下を決める:
- 予定コミット数
- 各コミットに入れるファイル/変更範囲
- 各コミットの `type[/scope]`

---

### 3) ステージング
#### 3.1 意図しない差分を除外
- ローカル設定・秘密情報（`.env*`）・生成物（`build/bin/`, `frontend/node_modules/`, `frontend/dist/`）・テスト用PDF（`*.pdf`）はステージしない

#### 3.2 ファイル単位でステージング
```bash
git add path/to/file.go
git add frontend/src/App.svelte
```

新規ファイルは内容を確認してから add する。

---

### 4) ステージ内容を最終確認（毎コミット必須）
```bash
git diff --cached --stat
git diff --cached
```

---

### 5) 検証（変更に応じて）
```bash
# フォーマット確認
go fmt ./...        # 差分が出たら format-only として別コミットに

# 静的解析・テスト
make lint
make test

# 依存を触ったら
make tidy           # 差分が出たら go.mod/go.sum をコミットに含める

# app.go のメソッドシグネチャ（Wails公開API）を変更したら
make generate        # frontend/wailsjs/* の差分を確認してコミットに含める

# フロントエンドを変更したら
cd frontend && pnpm install && pnpm exec svelte-check
```

CI（`.github/workflows/ci.yml`）は build / test / lint / fmt / tidy を実行するため、ローカルで通しておく。
`make build` / `make build-mac` はローカルで実際にビルドが通ることの確認にも使える。

---

### 6) コミットメッセージ生成（Conventional Commits）

フォーマット:
```
<type>[(<scope>)]: <subject>
```

**Types:** feat, fix, docs, style, refactor, perf, test, chore
**Scope:** 任意（例: `ai`, `cache`, `renamer`, `config`, `app`, `frontend`, `deps`, `ci`）

**Rules**
- Subject: 簡潔、末尾ピリオドなし、目安50文字以内、**日本語**
- Body: 必要な場合のみ（破壊的変更・移行・ロールバックは必ず明記）

**Examples:**
- `fix(ai): 金額抽出プロンプトの優先順位を修正`
- `feat(app): キャッシュを無視した再解析モードを追加`
- `chore(deps): 依存を更新`
- `chore(ci): GitHub Actions ワークフローを更新`

---

### 7) コミット実行（毎コミット）
```bash
git commit -m "type(scope): subject"
```

---

### 8) すべての変更がコミットされるまで繰り返す
```bash
git status --porcelain=v1
```

---

### 9) 完了条件
```bash
git log --oneline -n 10
git status
```

- working tree clean
- コミット粒度が適切
- メッセージに「何を/なぜ」がある（必要に応じて）

---

## Safety Stops（必ず止める）
- シークレット/鍵/トークンっぽい文字列が差分にある（`.env*`, APIキー等）
- `main` に直接コミットしようとしている（ユーザーが明示的に許可しない限り）
- コミット分割が必要なのに1コミットに押し込もうとしている
- `app.go` のWails公開APIを変更したのに `frontend/wailsjs/*` を再生成せずコミットしようとしている
