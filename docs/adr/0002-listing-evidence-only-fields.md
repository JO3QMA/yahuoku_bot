---
status: accepted
---

# Field は ListingEvidence と ModelInvariant に限定し CatalogConfiguration を除外する

Supplement の Web 検索がメーカーの BTO 販売オプション（CatalogConfiguration）を拾い、出品記載と無関係な CPU・メモリ等が Field に入っていた。Product の Field はこの Auction の出品内容を表すべきであり、カタログ上の選択肢は含めない。

**原則（全 Category 共通）**

- **InstalledConfiguration**（搭載 CPU・メモリ・ディスク等）: ListingEvidence（タイトル・説明文・出品画像）に裏付けがある場合のみ Field に入れる。曖昧な記載（例: 「HDD付き」のみ）は記載範囲だけ入れ、詳細は Supplement で補完しない。無ければ空欄。
- **CatalogConfiguration**（BTO 選択肢・標準構成・最大搭載数など）: Field に入れない。
- **ModelInvariant**（型番から導ける固定的仕様。例: 3.5インチベイ対応、対応 CPU 世代）: ListingEvidence が無くても Supplement で入れてよい。
- **識別**（型番・機種名の同定）: Supplement で補完可。

**server の Field 配置**

- `server_model`: 識別（Supplement 可）
- `storage_info`: 搭載ディスクは InstalledConfiguration。出品記載が無い場合はベイサイズ等の ModelInvariant のみ Supplement 可
- `cpu_model_line`: 搭載 CPU 型番のみ（ListingEvidence のみ）
- `other_notes`: 他 Field に収まらない ModelInvariant（例: 対応 CPU 世代）

**実装方針**

プロンプト強化に加え、Category ごとに Supplement で埋めてよい Field キーを Go 側で定義し、検索補完（`lookup_spec`）の対象を制限する。InstalledConfiguration 系キーは unresolved でも検索ループに入れない。

用語は `CONTEXT.md`（ListingEvidence, CatalogConfiguration, ModelInvariant, InstalledConfiguration）を参照。
