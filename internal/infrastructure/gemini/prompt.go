package gemini

import (
	"fmt"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func buildExtractPrompt(title, plainDesc string) string {
	var b strings.Builder
	b.WriteString(`以下のヤフオク商品のタイトルと説明文から、商品ジャンルを1つ判別し、該当ジャンルのテンプレート項目のみを抽出してください。

【ジャンル一覧（category に設定する値）】
`)
	for _, cat := range product.AllCategories {
		keys := product.TemplatesFor(cat)
		keyStrs := make([]string, len(keys))
		for i, d := range keys {
			keyStrs[i] = d.Key
		}
		fmt.Fprintf(&b, "- %s (%s): フィールドキー [%s]\n", cat, cat.DisplayName(), strings.Join(keyStrs, ", "))
	}

	b.WriteString(`
【判別ルール】
- 単体のCPU・メモリ・GPU・SSD/HDD・NICは該当パーツジャンル（cpu, memory, gpu, storage, nic）を優先
- ラックマウントレール単体は rack_rail、ラック本体は server_rack
- サーバー完成品・複合サーバー機器は server
- デスクトップPC・NUC・ミニPCは desktop_nuc
- スイッチ・ルータ・APは network（device_type に SW/RT/AP を記載）
- UPSは ups
- 分類できない場合は other

【server ジャンルの value 形式例】
- server_model: Fujitsu Primergy RX1330 M4
- cpu_model_line: Intel Core Ultra 7 355 @4.25GHz x1
- memory_info: DDR4 Unbuffered 2133MHz 8GB x8 Total: 64GB
- storage_info: SSD 256GB x1
- gpu: AMD Radeon RX9070XT x2（GPU非搭載の場合は fields に含めない）

【出力形式】
- category: 上記いずれか1つ
- condition: "新品" / "中古" / "不明"
- shipping_free: 送料無料なら true、落札者負担なら false
- fields: 判別したジャンルのフィールドキーのみ { "key": "...", "value": "..." } の配列。不明な項目は含めないか value を空に

【重要】
- タイトルにスペックが含まれることが多いため、タイトルを特に重視してください
- 商品説明が空の場合はタイトルのみから抽出してください
- fields の key は選択したジャンルのテンプレートキーのみ使用してください

【タイトル】
`)
	b.WriteString(title)
	b.WriteString("\n\n【商品説明】\n")
	b.WriteString(plainDesc)
	return b.String()
}
