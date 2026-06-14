package product

// Field は Product のスペック欄1項目（key と value）。
type Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Product は Extraction で得られた Auction 上の売り物情報。
type Product struct {
	Category     Category `json:"category"`
	Condition    string   `json:"condition"`
	FreeShipping *bool    `json:"shipping_free"`
	Fields       []Field  `json:"fields"`
}

// EmptyProduct は Extraction 失敗時のフォールバック Product。
func EmptyProduct() *Product {
	return &Product{Category: CategoryOther, Fields: nil}
}

// IsEffectivelyEmpty は実質的に Extraction 結果が空かどうかを返す。
func (p *Product) IsEffectivelyEmpty() bool {
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
