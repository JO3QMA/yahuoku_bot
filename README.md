# yahoo_auctions_bot

Discord 上でヤフオク（Yahoo! オークション）の URL に反応し、Auction 情報と Category 別 Field の Preview を Embed で返す Bot です。別リポジトリの **yahoo_auctions API**（Connect RPC）と **OpenAI 互換 API**（Chat Completions）を組み合わせて動作します。

Product 情報は **多段 Extraction パイプライン**（テキスト分類 → 画像解析 → 不足時のみ Web 検索 → 統合）で導出します。

## できること

- **メッセージ監視**: メッセージ内のヤフオク URL（`page.auctions.yahoo.co.jp/.../auction/<id>`）を検出し、Category（サーバー・GPU・NIC 等）を判別して FieldTemplate に沿った Preview を Embed で投稿します。
- **Watch**: Bot が投稿した Preview に 🔔（ベル）リアクションを付けると、その Auction の Watch を登録します（バックグラウンドで定期的にチェック）。
- **プレビュー CLI**（開発用）: オークション ID を渡すと JSON で Preview 結果を標準出力します（`cmd/preview`）。

## 必要なもの

- **Go** 1.25.4 以降（`go.mod` に準拠）
- **Discord Bot トークン**（`DISCORD_TOKEN`）
- **OpenAI 互換 API キー**（`OPENAI_API_KEY`。ベース URL は `OPENAI_BASE_URL` で変更可）
- **yahoo_auctions API** が `API_ENDPOINT` で参照できること
- **rqlite**（Watch 永続化。Compose または単体起動）

## クイックスタート

### 1. 環境変数

`.env.example` を参考に `.env` を作成します（[direnv](https://direnv.net/) 等での読み込みを想定）。

必須:

- `DISCORD_TOKEN`
- `OPENAI_API_KEY`

主な任意項目:

| 変数 | 説明 | 既定 |
|------|------|------|
| `OPENAI_BASE_URL` | OpenAI 互換 API のベース URL | `https://api.openai.com/v1` |
| `OPENAI_MODEL` | Stage1/4 用モデル | `gpt-4o-mini` |
| `OPENAI_MODEL_VISION` | Stage2 画像解析用モデル | `gpt-4o` |
| `OPENAI_MODEL_AGENT` | Stage3 エージェント補完用モデル | `gpt-4o` |
| `OPENAI_MAX_IMAGES` | 推論に使う最大画像数 | `3` |
| `OPENAI_MAX_SEARCH_CALLS` | 1商品あたりの最大 Web 検索回数 | `3` |
| `OPENAI_PIPELINE_TIMEOUT_SEC` | 多段推論パイプラインのタイムアウト（秒） | `45` |
| `API_ENDPOINT` | オークション API のベース URL | `http://localhost:8080` |
| `RQLITE_URL` | rqlite のベース URL | `http://localhost:4001` |
| `ALLOWED_GUILDS` | 反応するサーバー ID（カンマ区切り、空=全許可） | （空） |
| `ALLOWED_CHANNELS` | 反応するチャンネル ID（カンマ区切り、空=全許可） | （空） |
| `CHECK_INTERVAL_MINUTES` | ウォッチのポーリング間隔（分） | `5` |
| `POLL_DELAY_MS` | ポーリング時の 1 件あたりディレイ（ms） | `2000` |

### 2. ローカル実行（rqlite + Bot）

rqlite を起動してから Bot を実行します。

```bash
docker compose up rqlite -d
direnv allow   # direnv を使う場合
go run ./cmd/bot
```

yahoo_auctions API も必要な場合は別途起動するか、Compose 全体を使います。

### 3. Docker Compose（本番相当）

`.env` に `DISCORD_TOKEN` と `OPENAI_API_KEY` を設定します。`ALLOWED_CHANNELS` は `compose.yaml` に例示値があります。

```bash
make up
# または
docker compose up -d
```

`compose.yaml` では **yahoo_auctions API**・**rqlite**・本 Bot の 3 サービスが定義されています。

### 4. Git hooks（任意）

```bash
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

pre-commit では `go test ./...` を実行します。

## プレビュー CLI

オークション ID（URL の末尾の ID）を引数に取り、JSON を標準出力します。終了コード: `0` 成功、`1` 空のスペック、`2` エラー。

```bash
go run ./cmd/preview <auction_id>
```

## Makefile

| ターゲット | 内容 |
|------------|------|
| `make build` | `discord-bot` バイナリをビルド |
| `make run` | ビルド後に `./discord-bot` を実行 |
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run`（要インストール） |
| `make tidy` | `go mod tidy` |
| `make docker-build` | イメージ `yahoo-auctions-bot:local` をビルド |
| `make up` / `make down` | Compose の起動・停止 |

## アーキテクチャとエージェント向けドキュメント

レイヤー構成・慣習・関連リポジトリは [**Agents.md**](Agents.md) を参照してください。
