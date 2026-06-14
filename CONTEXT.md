# yahoo_auctions_bot

Discord 上のヤフオク URL に反応し、オークション情報のプレビュー表示と価格監視通知を行う Bot。

## Language

**Auction**:
ヤフオク上の1出品単位。`auctionID` で識別され、現在価格・終了時刻・ステータスなどのオークション属性を持つ。
_Avoid_: Listing, Item, 出品

**Product**:
Auction に載っている売り物そのもの。Category・Condition・FreeShipping・Field など、Extraction で得られた物品情報を指す。
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

**Condition**:
Product の物理・使用状態（新品、中古など）。Category や Field とは独立した Product 属性。
_Avoid_: 商品状態（日本語説明向け）, ProductAttribute

**FreeShipping**:
Product が送料無料かどうか。
_Avoid_: 送料無料（日本語説明向け）

**Extraction**:
Auction のタイトル・説明文・画像から Product を導き出す処理。
_Avoid_: Inference, Analysis, 推論（実装・モデル寄りの説明向け）

**Preview**:
Auction のオークション属性と Product の Extraction 結果を統合した、Discord 表示用データ。
_Avoid_: Embed（Discord の表示形式。presentation 層の用語）, AuctionSummary, 概要

**Watch**:
ユーザーが特定の Auction を価格・終了時刻まで追跡する登録。🔔 リアクションで登録し、ポーリングにより変動を通知する。機能全体も個別レコードも Watch と呼ぶ。
_Avoid_: Subscription, Alert, 監視アイテム

**PriceAlert**:
Watch 中の Auction で価格が最後に記録した値から上昇したときに送る通知。
_Avoid_: PriceIncreaseNotification, 値上がり通知（日本語説明向け）

**EndingReminder**:
Watch 中の Auction の終了が近づいたときに送る通知。1 Watch につき1回。
_Avoid_: EndingSoonNotification, 終了間近通知（日本語説明向け）, Reminder（単体では曖昧）

**NotificationThread**:
Preview メッセージに紐づく通知専用の Discord スレッド。同一メッセージの Watch は1つの NotificationThread を共有する。
_Avoid_: WatchThread, Thread（単体では曖昧）
