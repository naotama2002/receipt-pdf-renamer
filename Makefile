.PHONY: dev build build-mac build-win release-mac release-win clean test fmt lint tidy generate

# 開発モード
dev:
	wails dev

# ビルド
build:
	wails build

# macOS用ビルド（Universal Binary）
build-mac:
	wails build -platform darwin/universal

# Windows用ビルド
build-win:
	wails build -platform windows/amd64

# macOS用リリースビルド（最適化）
release-mac:
	wails build -platform darwin/universal -ldflags="-s -w"

# Windows用リリースビルド（最適化）
release-win:
	wails build -platform windows/amd64 -ldflags="-s -w"

# クリーン
clean:
	rm -rf ./build/bin
	rm -rf frontend/node_modules
	go clean

# テスト
test:
	go test -v -tags=test ./...

# フォーマット
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run

# 依存関係整理
tidy:
	go mod tidy

# Wails生成（バインディング更新）
generate:
	wails generate module
