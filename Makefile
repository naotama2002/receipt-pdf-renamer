.PHONY: build install clean test fmt lint tidy run

BINARY_NAME=receipt-pdf-renamer
BIN_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/receipt-pdf-renamer

install:
	go install $(LDFLAGS) ./cmd/receipt-pdf-renamer

clean:
	rm -rf $(BIN_DIR)
	go clean

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

run: build
	$(BIN_DIR)/$(BINARY_NAME)
