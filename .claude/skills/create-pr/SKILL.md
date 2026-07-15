---
name: create-pr
description: "Create a pull request from current branch with a high-quality title/body, ensuring commits are appropriately scoped and checks are run. ONLY use when user explicitly invokes /create-pr slash command. Do NOT trigger on general PR questions."
---

# Create PR SKILL

このSKILLは **「適切なブランチ・適切なコミット粒度・必要な検証を満たした状態で、レビューしやすいPRを作る」** ために使う。
ユーザーが **`/create-pr` を明示的に呼んだときだけ** 実行する。

このプロジェクト（receipt-pdf-renamer）は **Wails v2 + Svelte** 製のGUIアプリ（領収書PDFを解析してリネームするツール）。
- バックエンド: `main.go`, `app.go`, `internal/ai`, `internal/cache`, `internal/config`, `internal/renamer`, `internal/history`, `internal/pdf`
- フロントエンド: `frontend/src/*.svelte`（Svelte + TypeScript, pnpm）、生成バインディング `frontend/wailsjs/*`

---

## General Rules
- PR タイトル・本文は **日本語** で書く
- **シークレット/内部情報（APIキー、Keychain/Credential Managerの値、社内URL）を PR 本文に書かない**
- 目的が複数なら、PRを分ける（または「後続PRに分割」を提案）
- CIが落ちる見込みがある場合は理由と影響範囲を明記する

---

## Workflow

### 0) Preflight: ブランチと差分確認（必須）
```bash
git rev-parse --abbrev-ref HEAD
git status --porcelain=v1
git log --oneline --decorate -n 20
```

- ブランチが `main` の場合: **PRは作らず** 新規ブランチを作る
  - `feat/short-description` / `fix/short-description` / `chore/short-description` 等
- 作業ツリーが汚れている（未コミットがある）場合: **先にコミット**（`/git-commit` を案内）

---

### 1) リモート追従・差分レンジを確認
```bash
git fetch origin
git status -sb
git diff --stat origin/main...HEAD
```

---

### 2) PRのスコープが適切かチェック
- PR が大きすぎないか（レビュー可能サイズか）
- `go fmt` のみの変更が機能変更と混ざっていないか
- バックエンドAPI（`app.go`）変更に対応する `frontend/wailsjs/*` の再生成が含まれているか
- 破壊的変更が含まれるか（含むなら影響範囲・ロールバック必須）
- UI・フロントエンドの変更がある場合、`wails dev` 等で実際に動作確認したか

---

### 3) 検証（可能な範囲で）
```bash
go fmt ./...
make lint
make test
make tidy && git diff --exit-code go.mod go.sum   # 差分なしを確認

# フロントエンドに変更がある場合
cd frontend && pnpm install && pnpm exec svelte-check

# ビルドが通ることの確認（Wails CLIが必要）
make build
```

CI（`.github/workflows/ci.yml`）は build / test / lint / fmt / tidy を実行する。
実行できない検証（例: 実機でのUI動作確認、AI APIキーを使った解析）がある場合は「未実施」と明記する。

---

### 4) PR タイトル・本文を生成

#### 4.1 タイトル規約（Conventional Commits 形式）
```
<type>[(<scope>)]: <subject>
```

- 例: `fix(ai): 金額抽出プロンプトの優先順位を修正`
- 例: `feat(app): キャッシュを無視した再解析モードを追加`
- 例: `chore(deps): 依存を更新`
- 50〜72文字目安、簡潔に「何をするPRか」を日本語で

#### 4.2 PR 本文テンプレ
```md
## 背景

<なぜこの変更が必要か。チケットへのリンクがあれば貼る>

## やったこと

-

## テスト計画

- [ ]
```

**必須で書くべきもの:**
- 変更の意図（なぜ）
- 変更内容の箇条書き
- 確認方法（`make test` / `make lint` の結果、`wails dev` での動作確認、対象PDFでの解析結果など）

**省略可:**
- 変更が自明な場合はセクションを減らして良い
- 小さな変更（1ファイル・数行）は本文なしでも可

---

### 5) PR 作成（GitHub CLI）

リモートに push してから作成:
```bash
git push -u origin HEAD
```

PR 作成:
```bash
gh pr create \
  --base main \
  --head "$(git rev-parse --abbrev-ref HEAD)" \
  --title "type(scope): subject" \
  --body "$(cat <<'EOF'
## 背景

...

## やったこと

- ...

## テスト計画

- [ ] ...
EOF
)"
```

ドラフトで作る場合:
```bash
gh pr create --draft ...
```

---

### 6) PR 作成後チェック
```bash
gh pr view
```

- CI が走っているか
- 期待した差分/コミットが含まれているか

---

## Output Requirements（このSKILLの最終出力）
- PR タイトル案
- PR 本文案（上記テンプレに沿う）
- 実行するコマンド（`gh pr create ...`）
- 未実施の検証がある場合は、その明記

---

## Safety Stops（必ず止める）
- 未コミット変更があるのにPRを作ろうとしている
- シークレット/APIキー/鍵らしき情報が差分にある（`.env*` 等）
- `main` ブランチから直接PRを作ろうとしている
- `app.go` のWails公開APIを変更したのに `frontend/wailsjs/*` の再生成が含まれていない
