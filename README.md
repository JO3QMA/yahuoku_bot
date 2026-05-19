# yahoo_auctions_bot

Discord 上でヤフオク（Yahoo! オークション）の商品 URL に反応し、オークション情報と商品ジャンル別スペックのプレビューを Embed で返す Bot です。別リポジトリの **yahoo_auctions API**（Connect RPC）と **Google Gemini** を組み合わせて動作します。

## できること

- **メッセージ監視**: メッセージ内のヤフオク URL（`page.auctions.yahoo.co.jp/.../auction/<id>`）を検出し、商品ジャンル（サーバー・GPU・NIC 等）を判別してテンプレートに沿った情報を Embed で投稿します。
- **ウォッチ**: Bot が投稿したプレビューに 🔔（ベル）リアクションを付けると、そのオークションを監視リストに登録します（バックグラウンドで定期的にチェック）。
- **プレビュー CLI**（開発用）: オークション ID を渡すと JSON でプレビュー結果を標準出力します（`cmd/preview`）。

## 必要なもの

- **Go** 1.25.4 以降（`go.mod` に準拠）
- **Discord Bot トークン**（`DISCORD_TOKEN`）
- **Gemini API キー**（`GEMINI_API_KEY`）
- **yahoo_auctions API** が `API_ENDPOINT` で参照できること（ローカル・Docker のいずれでも可）

## クイックスタート

### 1. 設定ファイル

`config.yaml.example` をコピーして `config.yaml` を用意し、`allowed.guilds` / `allowed.channels` を必要に応じて編集します。いずれかのリストが空の場合は、その軸では制限しません（全サーバーまたは全チャンネル許可）。

```bash
cp config.yaml.example config.yaml
```

### 2. 環境変数

`.env.example` を参考に `.env` を作成します（[direnv](https://direnv.net/) 等での読み込みを想定）。

必須:

- `DISCORD_TOKEN`
- `GEMINI_API_KEY`

主な任意項目:

| 変数 | 説明 | 既定 |
|------|------|------|
| `GEMINI_MODEL` | 使用する Gemini モデル | `gemini-2.5-flash-lite` |
| `API_ENDPOINT` | オークション API のベース URL | `http://localhost:8080` |
| `CONFIG_PATH` | `config.yaml` のパス（Bot・`cmd/preview` の両方で参照） | `config.yaml` |
| `RQLITE_URL` | 設定時は **rqlite** に接続（本番・Compose 向け）。未設定時は SQLite | （未設定） |
| `DB_PATH` | SQLite のパス（`RQLITE_URL` 未設定時のみ） | `data/watch.db` |
| `CHECK_INTERVAL_MINUTES` | ウォッチのポーリング間隔（分） | `5` |
| `POLL_DELAY_MS` | ポーリング時の 1 件あたりディレイ（ms） | `2000` |

### 3. ローカル実行

オークション API を起動したうえで:

```bash
direnv allow   # direnv を使う場合
go run ./cmd/bot
```

### 4. Docker Compose

`.env` に `DISCORD_TOKEN` と `GEMINI_API_KEY` を設定し、`API_ENDPOINT` は `.env.example` の例のとおり Compose 内のサービス名に合わせてください。

```bash
make up
# または
docker compose up -d
```

`compose.yaml` では **yahoo_auctions API**・**rqlite**・本 Bot の 3 サービスが定義されています。

## プレビュー CLI

オークション ID（URL の末尾の ID）を引数に取り、JSON を標準出力します。終了コード: `0` 成功、`1` 空のスペック、`2` エラー。

```bash
export CONFIG_PATH=config.yaml   # 任意
go run ./cmd/preview <auction_id>
```

## Makefile

| ターゲット | 内容 |
|------------|------|
| `make build` | `discord-bot` バイナリをビルド |
| `make run` | ビルド後に `./discord-bot` を実行 |
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run`（要インストール） |
| `make docker-build` | イメージ `yahoo-auctions-bot:local` をビルド |
| `make up` / `make down` | Compose の起動・停止 |

## アーキテクチャとエージェント向けドキュメント

レイヤー構成・慣習・関連リポジトリは [**Agents.md**](Agents.md) を参照してください。
