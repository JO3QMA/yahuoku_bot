package gemini

import (
	"fmt"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func categoryBlock() string {
	var b strings.Builder
	for _, cat := range product.AllCategories {
		keys := product.TemplatesFor(cat)
		keyStrs := make([]string, len(keys))
		for i, d := range keys {
			keyStrs[i] = d.Key
		}
		fmt.Fprintf(&b, "- %s (%s): フィールドキー [%s]\n", cat, cat.DisplayName(), strings.Join(keyStrs, ", "))
	}
	return b.String()
}

func classificationRules() string {
	return `
【判別ルール】
- 単体のCPU・メモリ・GPU・SSD/HDD・NICは該当パーツジャンル（cpu, memory, gpu, storage, nic）を優先
- ラックマウントレール単体は rack_rail、ラック本体は server_rack
- サーバー完成品・複合サーバー機器は server
- デスクトップPC・NUC・ミニPCは desktop_nuc
- スイッチ・ルータ・APは network（device_type に SW/RT/AP を記載）
- UPSは ups
- 分類できない場合は other
`
}

func serverValueExamples() string {
	return `
【server ジャンルの value 形式例】
- server_model: Fujitsu Primergy RX1330 M4
- cpu_model_line: Intel Core Ultra 7 355 @4.25GHz x1
- memory_info: DDR4 Unbuffered 2133MHz 8GB x8 Total: 64GB
- storage_info: SSD 256GB x1
- gpu: AMD Radeon RX9070XT x2（GPU非搭載の場合は fields に含めない）
`
}

func buildStage1Prompt(title, plainDesc string) string {
	var b strings.Builder
	b.WriteString(`以下のヤフオク商品のタイトルと説明文から、商品ジャンルを1つ判別し、該当ジャンルのテンプレート項目を抽出してください。
また、テンプレート上まだ埋められていないフィールドキーを missing_keys に、Web検索で補完できそうなクエリを candidate_queries に列挙してください。

【ジャンル一覧（category に設定する値）】
`)
	b.WriteString(categoryBlock())
	b.WriteString(classificationRules())
	b.WriteString(serverValueExamples())
	b.WriteString(`
【出力形式】
- category, condition, shipping_free, fields（判別ジャンルのテンプレートキーのみ）
- missing_keys: 値が不明または空のテンプレートキー
- candidate_queries: 型番・スペック補完用の日本語検索クエリ（最大3件）

【重要】
- タイトルを特に重視してください
- 商品説明が空の場合はタイトルのみから抽出してください
- 確実に分かる項目だけ fields に入れ、推測で埋めないでください
- fields の key は選択したジャンルのテンプレートキーのみ使用してください

【タイトル】
`)
	b.WriteString(title)
	b.WriteString("\n\n【商品説明】\n")
	b.WriteString(plainDesc)
	return b.String()
}

func buildStage2Prompt(title, plainDesc string) string {
	var b strings.Builder
	b.WriteString(`添付されたヤフオク商品画像から、ラベル・型番・スペック表記など視認できる情報を抽出してください。

【対象ジャンル候補とフィールドキー】
`)
	b.WriteString(categoryBlock())
	b.WriteString(`
【出力】
- image_fields: 画像から読み取れた {key, value, confidence}（confidence は high/medium/low）
- visible_model_numbers: 画像に写っている型番・品番の配列

【重要】
- 読み取れない項目は含めない
- タイトル・説明文の参考情報:
  タイトル: `)
	b.WriteString(title)
	b.WriteString("\n説明: ")
	b.WriteString(plainDesc)
	return b.String()
}

func buildStage3Prompt(title, plainDesc string, s1 *stage1Result, s2 *stage2Result) string {
	var b strings.Builder
	b.WriteString(`不足している商品スペックを補完してください。必要なら lookup_spec 関数でWeb検索してください。
補完が完了したら done を true にし、fields にテンプレートキーと値を返してください。

【優先度】テキスト > 画像 > 検索結果

【タイトル】
`)
	b.WriteString(title)
	b.WriteString("\n\n【商品説明】\n")
	b.WriteString(plainDesc)
	b.WriteString("\n\n【Stage1 抽出】\n")
	b.WriteString(stageEvidenceJSON(s1))
	if s2 != nil {
		b.WriteString("\n\n【Stage2 画像解析】\n")
		b.WriteString(stage2EvidenceJSON(s2))
	}
	if s1 != nil && len(s1.MissingKeys) > 0 {
		b.WriteString("\n\n【補完対象キー】\n")
		b.WriteString(strings.Join(s1.MissingKeys, ", "))
	}
	return b.String()
}

func buildMergePrompt(title, plainDesc string, s1 *stage1Result, s2 *stage2Result, agentFields []product.Field, searchNotes []string) string {
	var b strings.Builder
	b.WriteString(`以下の証拠を統合し、最終的な商品情報を JSON で返してください。

【優先度】テキスト > 画像 > 検索

【タイトル】
`)
	b.WriteString(title)
	b.WriteString("\n\n【商品説明】\n")
	b.WriteString(plainDesc)
	b.WriteString("\n\n【Stage1】\n")
	b.WriteString(stageEvidenceJSON(s1))
	if s2 != nil {
		b.WriteString("\n\n【Stage2 画像】\n")
		b.WriteString(stage2EvidenceJSON(s2))
	}
	if len(agentFields) > 0 {
		b.WriteString("\n\n【Stage3 補完フィールド】\n")
		for _, f := range agentFields {
			fmt.Fprintf(&b, "- %s: %s\n", f.Key, f.Value)
		}
	}
	if len(searchNotes) > 0 {
		b.WriteString("\n\n【検索メモ】\n")
		for _, n := range searchNotes {
			b.WriteString(n)
			b.WriteString("\n")
		}
	}
	b.WriteString(`
【出力】category, condition, shipping_free, fields（判別ジャンルのテンプレートキーのみ）
`)
	return b.String()
}

func stageEvidenceJSON(s1 *stage1Result) string {
	if s1 == nil {
		return "{}"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "category: %s\ncondition: %s\n", s1.Category, s1.Condition)
	for _, f := range s1.Fields {
		fmt.Fprintf(&b, "- %s: %s\n", f.Key, f.Value)
	}
	return b.String()
}

func stage2EvidenceJSON(s2 *stage2Result) string {
	if s2 == nil {
		return "{}"
	}
	var b strings.Builder
	for _, f := range s2.ImageFields {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", f.Key, f.Value, f.Confidence)
	}
	if len(s2.VisibleModelNumbers) > 0 {
		b.WriteString("型番: ")
		b.WriteString(strings.Join(s2.VisibleModelNumbers, ", "))
		b.WriteString("\n")
	}
	return b.String()
}

// buildExtractPrompt は後方互換テスト用に Stage1 プロンプトのエイリアス。
func buildExtractPrompt(title, plainDesc string) string {
	return buildStage1Prompt(title, plainDesc)
}
