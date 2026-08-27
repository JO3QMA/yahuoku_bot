---
status: accepted
---

# 商品取得を yahoo_auctions API から sansai ライブラリへ移行する

Connect RPC の別サービス `yahoo_auctions_api` を廃止し、`github.com/jo3qma/sansai` を in-process で呼ぶ。3 C2C マーケット（ヤフオク・Yahoo!フリマ・メルカリ）に対応し、ドメインの中心概念を **Listing**（`market` + `listingID`）に置く。**Auction** は Market ではなく販売形式（入札・終了時刻あり）として定義する。

**段階:** sansai 側の正規化（[#6](https://github.com/JO3QMA/sansai/issues/6) / PR #8 マージ済み）が完了したため、Bot 移行 PR に着手できる。リポジトリ名の変更は移行完了後の別 PR とする。

## Considered Options

| 案 | 却下理由 |
|----|----------|
| yahoo_auctions API 維持 | 別コンテナ・protobuf 依存が残り、3 マーケット対応のたびに API 側も拡張が必要 |
| sansai CLI を subprocess で呼ぶ | JSON パースのオーバーヘッドとプロセス管理が不要。sansai はライブラリ利用を想定している |
| Bot 先行・sansai 拡張は後追い | メルカリオークションの EndingReminder 等が不完全なまま出荷される |
| Watch DB を `market` 列追加で移行 | 破壊的変更で十分。既存 Watch 件数は少なく、スキーマを `listing_id` に揃える方が明快 |

## Consequences

- `compose.yaml` から `yahoo-auctions-api` を削除。`API_ENDPOINT`・`connectrpc`・`protobuf` 依存も外す。
- `watch_items` は作り直し（`market` + `listing_id`）。既存 Watch は放棄する。
- Watch は 3 Market 共通。`PriceAlert` は価格変動全般。`EndingReminder` は Auction のみ。
- URL 検出は各マーケットの公式ドメイン代表パターンのみ。非対応 URL・取得失敗は無言スルー（ログのみ）。
- Preview Embed は Market 名を常時表示。販売形式は Auction のときのみ「オークション」を出す。
- PollingWorker の終了判定は sansai の `is_active` に委ね、Market 別ステータス文字列は Bot 側に持たない。
