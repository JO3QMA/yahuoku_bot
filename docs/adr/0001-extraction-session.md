---
status: accepted
---

# Extraction を単一セッション + Supplement に移行する

Extraction は Stage 1–4 の独立 API 呼び出しから、Go ミラー付き session 実装に置き換えた。メイン推論と Supplement（画像解析・Web 検索）で Product を組み立て、UnresolvedField の解消は Go が保証し、値は LLM と evidence に委ねる。終了は合意型（`done:true` は unresolved が空のときのみ承認）とし、上限到達時は部分 Product を返す。画像 Supplement は Extraction 開始と並列起動し、失敗はベストエフォートとする。旧 pipeline は削除済み。
