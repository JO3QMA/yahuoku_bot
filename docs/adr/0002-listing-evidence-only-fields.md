---
status: accepted
---

# Field は ListingEvidence と ModelInvariant に限定し CatalogConfiguration を除外する

Supplement の Web 検索が BTO 販売オプション（CatalogConfiguration）を Field に入れていた。Product の Field は出品内容（ListingEvidence）と型番固有仕様（ModelInvariant）に限定する。

`SupplementEligibleKeys` で検索補完と `applySupplementFields` の対象キーを制限し、InstalledConfiguration 系は unresolved でも検索ループに入れない。用語定義は `CONTEXT.md` を参照。
