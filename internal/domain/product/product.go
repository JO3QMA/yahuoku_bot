package product

// Field はテンプレート1項目のキーと値。
type Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ProductDetail はGeminiが抽出した商品情報。
type ProductDetail struct {
	Category     Category `json:"category"`
	Condition    string   `json:"condition"`
	ShippingFree *bool    `json:"shipping_free"`
	Fields       []Field  `json:"fields"`
}

// EmptyProductDetail は抽出失敗時のフォールバック。
func EmptyProductDetail() *ProductDetail {
	return &ProductDetail{Category: CategoryOther, Fields: nil}
}

// IsEffectivelyEmpty は実質的に抽出結果が空かどうかを返す。
func (p *ProductDetail) IsEffectivelyEmpty() bool {
	if p == nil {
		return true
	}
	if p.Category != CategoryOther && p.Category != "" {
		return false
	}
	if p.Condition != "" {
		return false
	}
	for _, f := range p.Fields {
		if f.Value != "" && f.Value != "不明" {
			return false
		}
	}
	return true
}
