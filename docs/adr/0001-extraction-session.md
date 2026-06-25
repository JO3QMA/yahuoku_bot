---
status: accepted
---

# Extraction を単一セッション + Supplement に移行する

Extraction はこれまで Stage 1–4 の独立 API 呼び出しで Product を組み立てていたが、Stage 間の JSON 再埋め込みにより文脈が断ち切られ、画像・検索・テキストの齟齬が起きうる。品質と実装の単純化のため、メインは単一 LLM 会話（1 本の推論スレッド）とし、画像解析と Web 検索は Supplement（サブエージェント）としてツール経由で呼ぶ。Product ドラフトは Go 側にミラーし、FieldTemplate の構造と UnresolvedField の解消は Go が保証し、Field の値は LLM と Supplement の evidence に委ねる。終了は合意型（LLM の done は Go が unresolved が空のときのみ承認）とし、検索上限・タイムアウトで打ち切った場合は部分 Product を返す。画像があるときは analyze_images を Extraction 開始と並列起動する。移行は `GEMINI_EXTRACTION_MODE=session|pipeline`（既定 pipeline）で共存させ、Preview CLI で session を先に検証してから Bot を切り替え、安定後に pipeline を削除する。Supplement の失敗はベストエフォートとし、得られた evidence だけで Extraction を続行する。

## Considered Options

- **4 段パイプライン維持**: 実装は既知だが Stage 間で文脈が再シリアライズのみで、品質目標に届かない。
- **単一 LLM のみ（Supplement なし）**: 実装は最も単純だが Vision / 検索用モデルを統合できず、品質とコストのトレードオフが悪い。
- **Go オーケストレーターのみ（LLM 会話なし）**: Go は薄くなるが、メイン推論の一貫性が LLM 会話より劣る。
- **一括置換**: diff は小さいが Preview 本番を比較なしに切り替えるリスクがある。

## Consequences

- `internal/infrastructure/gemini/` に session 実装が並び、しばらく pipeline と二重管理になる。
- Preview CLI が session モードの最初の検証口になる。
- glossary に UnresolvedField・Supplement を追加済み。実装語の「サブエージェント」「Stage」はコード内に留める。
