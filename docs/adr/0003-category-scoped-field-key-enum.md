---
status: accepted
---

# Field キーはカテゴリ固定で正規化する

LLM が `server` 向けに `memory_capacity`・`storage_type` などテンプレート外の `fields[].key` を返し、`ValidateFields` で捨てられていた。`server` の Field は `memory_info` / `storage_info` のまとめ形式を維持する（InstalledConfiguration の粒度は変えない）。

**決定:** `fieldKeyAliases` は使わず、正規化は Category 固定の FieldTemplate キーに一本化する。`finalize` と supplement では `mirror.category` をプロンプトで固定し、該当 Category の FieldTemplate キーのみを `fields[].key` として受け付ける。Stage 1 / Stage 2 は category 未確定または並列のためキーを制約せず、未知キーは捨てるが `[product] unknown field key` を警告ログする（タイトル・説明文は merge プロンプトに残る）。

> Gemini 実装では JSON Schema の enum で出力を制約していた。OpenAI 互換 API（`json_object` モード）へ移行後は、プロンプトによる制約と `CanonicalFieldKey` による事後検証・警告で同じ意図を実現する。

**却下した案:** alias 拡張（`memory_capacity`→`memory_info` 等）、Stage 1 を category 判別と fields 抽出に分割して全ステージでキーを固定、全カテゴリのキー union enum。
