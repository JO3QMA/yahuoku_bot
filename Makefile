# yahoo_auctions_bot — Makefile

BINARY_NAME := discord-bot
PKG_PATH   := ./cmd/bot
LDFLAGS   := -s -w

.PHONY: build run test lint tidy clean docker-build up down

# バイナリをビルドする（ローカル用）
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(PKG_PATH)

# Bot を起動する（direnv で .env を読み込み、rqlite が必要）
run: build
	./$(BINARY_NAME)

# テストを実行
test:
	go test ./...

# golangci-lint で静的解析
lint:
	golangci-lint run ./...

# 依存関係を整理
tidy:
	go mod tidy

# ビルド成果物を削除
clean:
	rm -f $(BINARY_NAME)

# このリポジトリの Docker イメージをビルド
docker-build:
	docker build -t yahoo-auctions-bot:local .

# Compose で API + Bot を起動（環境変数は .env を direnv 等で読み込むか compose で指定）
up:
	docker compose up -d

# Compose で停止
down:
	docker compose down
