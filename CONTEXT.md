# yahoo_auctions_bot

Discord 上の C2C マーケット（ヤフオク・Yahoo!フリマ・メルカリ）URL に反応し、出品情報の Preview 表示と価格監視通知を行う Bot。

## Language

**Listing**:
いずれかの C2C マーケット上の1出品単位。`market` と `listingID` の組で識別される。Preview・Watch・Extraction の対象。
_Avoid_: Item（sansai の取得型）, 出品, Posting

**Market**:
Listing が属する C2C サービス。`yahoo_auction`・`yahoo_flea`・`mercari` のいずれか。
_Avoid_: Platform, Site, マーケットプレイス

**Auction**:
入札により価格が変動し、終了時刻を持つ Listing。`yahoo_auction` Market の出品、および `mercari` Market のオークション形式出品が該当する。Market とは独立した販売形式。
_Avoid_: オークション出品（日本語説明向け）, BidSession

**FixedPriceListing**:
固定価格で販売される Listing。`yahoo_flea` Market の出品、および `mercari` Market の通常（フリマ）出品が該当する。
_Avoid_: 定額出品, フリマ出品

**Product**:
Listing に載っている売り物そのもの。Category・Condition・FreeShipping・Field など、Extraction で得られた物品情報を指す。
_Avoid_: Spec, 商品詳細

**Category**:
Product の種別判別軸。`server`・`gpu`・`nic` などの値を持ち、FieldTemplate を決める。
_Avoid_: Genre, ジャンル（UI説明・日本語コメント向け）, ProductType

**FieldTemplate**:
Category ごとに定義されたスペック欄の項目定義。何を抽出・表示するかを決める（例: サーバーなら CPU・メモリ・ストレージ）。
_Avoid_: ジャンル別スペック（日本語説明向け）, SpecField

**Field**:
Product に実際に入ったスペック欄の1項目。key と value の組。
_Avoid_: Spec, Attribute, スペック値

**ListingEvidence**:
Listing のタイトル・説明文・出品画像から読み取れる事実。Field の根拠となる情報源。
_Avoid_: 出品情報, オークション記載

**CatalogConfiguration**:
メーカーが筐体などを販売する際に選択できる標準構成・オプション一覧（例: BTO の CPU・メモリ・搭載ディスクの選択肢）。この Listing に実際に載っている構成とは限らない。Field の値にならない。
_Avoid_: 販売オプション, カタログスペック, メーカー構成

**ModelInvariant**:
型番・機種名から導ける、筐体やプラットフォームの固定的な仕様（例: 3.5インチベイ対応、対応 CPU 世代・ソケット世代）。購入時の選択肢ではなく設計上の属性。最大搭載数などカタログ上の選択肢に近い数値は含まない。ListingEvidence に無くても Supplement で Field に入れてよい。
_Avoid_: カタログ構成, 搭載スペック, 最大搭載数

**InstalledConfiguration**:
この Listing の出品物に実際に搭載・装着されている構成（CPU 型番、メモリ容量、入っているディスクなど）。ListingEvidence に裏付けがある場合のみ Field に入る。曖昧な記載（例: 「HDD付き」のみ）は記載された範囲だけ入れ、詳細を Supplement で補完しない。`cpu_model_line`・`memory_info` 等の搭載スペック欄は原則ここに属する。
_Avoid_: 標準構成, 出荷時構成

**Condition**:
Product の物理・使用状態（新品、中古など）。Category や Field とは独立した Product 属性。
_Avoid_: 商品状態（日本語説明向け）, ProductAttribute

**FreeShipping**:
Product が送料無料かどうか。
_Avoid_: 送料無料（日本語説明向け）

**Extraction**:
Listing のタイトル・説明文・画像から Product を導き出す処理。
_Avoid_: Inference, Analysis, 推論（実装・モデル寄りの説明向け）

**UnresolvedField**:
FieldTemplate に定義されたキーのうち、Extraction 完了前の Product にまだ値が入っていないもの。
_Avoid_: missing_key, 未抽出フィールド（実装・プロンプト寄りの説明向け）

**Supplement**:
ListingEvidence に無い情報を画像解析や Web 検索で補う手段。
_Avoid_: Sub-agent, Tool, サブエージェント（実装寄りの説明向け）

**Preview**:
Listing のマーケット属性（価格・ステータス等）と Product の Extraction 結果を統合した、Discord 表示用データ。
_Avoid_: Embed（Discord の表示形式。presentation 層の用語）, AuctionSummary, 概要

**Watch**:
ユーザーが特定の Listing を追跡する登録。3 Market すべてが対象。🔔 / 👀 リアクションで登録し、ポーリングにより変動を通知する。機能全体も個別レコードも Watch と呼ぶ。
_Avoid_: Subscription, Alert, 監視アイテム

**PriceAlert**:
Watch 中の Listing で価格が最後に記録した値から変動したときに送る通知。入札による上昇（Auction）と出品者による値下げ・値上げ（FixedPriceListing）の両方を含む。
_Avoid_: PriceIncreaseNotification, 値上がり通知（日本語説明向け）

**EndingReminder**:
Watch 中の Auction の終了が近づいたときに送る通知。1 Watch につき1回。FixedPriceListing には送らない。
_Avoid_: EndingSoonNotification, 終了間近通知（日本語説明向け）, Reminder（単体では曖昧）

**NotificationThread**:
Preview メッセージに紐づく通知専用の Discord スレッド。同一メッセージの Watch は1つの NotificationThread を共有する。
_Avoid_: WatchThread, Thread（単体では曖昧）
