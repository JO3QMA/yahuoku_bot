# Agents.md — yahoo_auctions_bot

AIエージェントがこのリポジトリで作業する際のコンテキストと指針です。

---

## プロジェクト概要

**yahoo_auctions_bot** は、Discord上でヤフオク（Yahoo! オークション）の URL に反応し、Auction 情報と Category 別 Field の Preview を Embed で返す Discord Bot です。

- **入力**: Discordメッセージ内のヤフオクURL（`page.auctions.yahoo.co.jp/.../auction/<id>`）
- **処理**: 外部 API で Auction 取得 → 多段 Extraction パイプライン（分類・画像解析・オンデマンド検索・統合）→ Embed 生成
- **出力**: 同一チャンネルへ Embed 送信（現在価格・残り時間・Condition・送料・Category・Field など）

---

## アーキテクチャ

- **レイヤー**: クリーン/ヘキサゴナル寄りの構成
  - `cmd/bot` — エントリポイント。設定読み込み・DI・Bot起動
  - `internal/domain` — ドメイン仕様（例: `product.Product`, `watch.Watch`）
  - `internal/application` — ユースケース（例: `PreviewUsecase`）
  - `internal/infrastructure` — 外部サービス（オークションAPI Connect RPC、Gemini）
  - `internal/presentation` — Discord（arikawa 利用、ハンドラー・Embed）

- **依存の向き**: `cmd` → `internal/*`。infrastructure が application のインターフェースを実装し、presentation が application を呼ぶ。ドメインは他レイヤーに依存しない。

- **外部連携**
  - **yahoo_auctions API**: `API_ENDPOINT` の Connect RPC（`GetAuction`）。別リポジトリ `yahoo_auctions` が提供想定。
  - **protobuf**: `github.com/jo3qma/protobuf/gen/go` の生成コードを使用。APIの型・RPCはここに依存。
  - **Gemini**（`google.golang.org/genai`）: タイトル・説明・商品画像から `product.Product` を多段 Extraction パイプラインで導出。不足 Field は Function Calling 経由で Google Search をオンデマンド実行。
  - **Discord**: arikawa v3（Gateway、メッセージ、Embed送信）。

---

## 技術スタック・バージョン

- **Go**: 1.25.4（`go.mod` に準拠）
- **主要ライブラリ**: connectrpc.com/connect, arikawa/v3, google.golang.org/genai, gopkg.in/yaml.v3
- **設定**: 環境変数（direnv で `.env` を読み込む想定） + YAML（`config.yaml`）。`config.Load(configPath)` で統合。

---

## 設定

| 種別 | 項目 | 説明 |
|------|------|------|
| 環境変数 | `DISCORD_TOKEN` | 必須。Botトークン |
| 環境変数 | `GEMINI_API_KEY` | 必須。Gemini APIキー |
| 環境変数 | `GEMINI_MODEL` | 任意。Stage1/4 用モデル（未設定時 `gemini-2.5-flash-lite`） |
| 環境変数 | `GEMINI_MODEL_VISION` | 任意。Stage2 画像解析用（未設定時 `gemini-2.5-flash`） |
| 環境変数 | `GEMINI_MODEL_AGENT` | 任意。Stage3 エージェント補完用（未設定時 `gemini-2.5-flash`） |
| 環境変数 | `GEMINI_MAX_IMAGES` | 任意。Extraction に使う最大画像数（未設定時 `3`） |
| 環境変数 | `GEMINI_MAX_SEARCH_CALLS` | 任意。1 Product あたりの最大検索回数（未設定時 `3`） |
| 環境変数 | `GEMINI_PIPELINE_TIMEOUT_SEC` | 任意。多段 Extraction パイプラインのタイムアウト秒（未設定時 `45`） |
| 環境変数 | `API_ENDPOINT` | オークションAPIのベースURL（未設定時 `http://localhost:8080`） |
| 環境変数 | `CONFIG_PATH` | 任意。YAML設定パス（未設定時 `config.yaml`） |
| YAML | `allowed.guilds` | 空でなければ、ここに列挙したサーバーのみ反応 |
| YAML | `allowed.channels` | 空でなければ、ここに列挙したチャンネルのみ反応 |

サンプルは `.env.example`（direnv 用）と `config.yaml.example` を参照。

---

## ディレクトリ・ファイル慣習

- **パッケージ名**: ディレクトリ名と一致（`discord`, `auction`, `config`, `product`, `gemini` など）。
- **インターフェース**: ユースケースが依存する取得・Extraction などは application 内でインターフェース定義し、infrastructure が実装（例: `AuctionFetcher`, `Extractor`）。
- **コメント**: 公開型・関数は日本語で説明を付ける（「〜を返す」「〜を行う」など）。
- **エントリ**: 実行バイナリは `cmd/bot/main.go` からビルド。Dockerでは `./cmd/bot` をビルドし、`discord-bot` として実行。

---

## 開発・実行

- **ローカル**: direnv で `.env` を読み込み、オークションAPIが `API_ENDPOINT` で動いている状態で `go run ./cmd/bot`。
- **Docker**: `Dockerfile` はマルチステージビルド。`compose.yaml` があれば他サービス（例: オークションAPI）と一緒に起動する想定。
- **テスト**: テストコードが存在する場合は `go test ./...` で実行。追加時は `*_test.go` を同パッケージに配置。

---

## エージェント作業時の指針

1. **既存アーキテクチャを尊重する**  
   新機能は「どのレイヤーに属するか」を意識し、ドメインに他レイヤー依存を増やさない。

2. **依存関係の追加**  
   新規パッケージは `go.mod` を更新。必要に応じて `go mod tidy`。大きな方針変更はユーザーに確認する。

3. **設定の追加**  
   設定項目は `internal/config` の `Config` および YAML構造と整合させ、`.env.example`（direnv 用）/ `config.yaml.example` を更新する。

4. **API・protobufの変更**  
   オークションAPIの型・RPCを変える場合は `yahoo_auctions` および `protobuf` リポジトリとの整合を考慮する。

5. **エラーハンドリング**  
   ユースケースやinfrastructureでは、エラーを握りつぶさず呼び出し元に返す。presentationではログを残しつつ、ユーザーには必要に応じてメッセージで伝える。

6. **日本語コメント**  
   公開APIや複雑なロジックには日本語でコメントを付ける。

---

## 関連リポジトリ

- **yahoo_auctions**: オークション情報を返すAPI。本BotはそのConnect RPCクライアントとして動作。
- **protobuf**: Connect用の.proto定義と生成Goコード。本Botは `gen/go` を参照。

---

## Agent skills

### Issue tracker

Issues live in this repo's GitHub Issues (`JO3QMA/yahuoku_bot`). See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles map to GitHub labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

---

このファイルは、エージェントが yahoo_auctions_bot の目的・構成・慣習を把握し、一貫した変更や機能追加を行うための参照用です。
