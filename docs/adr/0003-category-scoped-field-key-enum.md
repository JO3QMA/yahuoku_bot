---
status: accepted
---

# Field キー正規化は alias ではなくカテゴリ固定 schema enum で行う

Gemini が `server` 向けに `memory_capacity`・`storage_type` などテンプレート外の `fields[].key` を返し、`ValidateFields` で捨てられていた。`server` の Field は `memory_info` / `storage_info` のまとめ形式を維持する（InstalledConfiguration の粒度は変えない）。

**決定:** `fieldKeyAliases` を削除し、正規化は JSON Schema の enum に一本化する。`finalize` と supplement（`agentFieldsSchema`）では `mirror.category` を固定し、その Category の FieldTemplate キーのみを `fields[].key` の enum にする。Stage 1 / Stage 2 は category 未確定または並列のため enum を付けず、未知キーは捨てるが `[product] unknown field key` を警告ログする（タイトル・説明文は merge プロンプトに残る）。

**却下した案:** alias 拡張（`memory_capacity`→`memory_info` 等）、Stage 1 を category 判別と fields 抽出に分割して全ステージ enum 化、全カテゴリのキー union enum。
