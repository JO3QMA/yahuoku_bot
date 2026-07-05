package market

import (
	"fmt"
	"math"
	"slices"
)

// MarketEstimate は類似 Product の想定価格帯と根拠の一言。
type MarketEstimate struct {
	LowPrice  int64  `json:"low_price"`
	HighPrice int64  `json:"high_price"`
	Note      string `json:"note"`
}

// DisplayValue は Discord Embed 用の表示文字列を返す。
func (m *MarketEstimate) DisplayValue() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("¥%s〜¥%s（%s）", formatIntWithComma(m.LowPrice), formatIntWithComma(m.HighPrice), m.Note)
}

// FromPrices は落札価格配列から 25〜75 パーセンタイルの MarketEstimate を生成する。
func FromPrices(prices []int64, note string) (*MarketEstimate, bool) {
	if len(prices) == 0 {
		return nil, false
	}
	sorted := append([]int64(nil), prices...)
	slices.Sort(sorted)
	low := percentile(sorted, 0.25)
	high := percentile(sorted, 0.75)
	// percentile(sorted, 0.25) <= percentile(sorted, 0.75) はソート済み配列で常に成立。
	// 将来 percentile 実装変更時の安全弁として残す。
	if low > high {
		low, high = high, low
	}
	return &MarketEstimate{LowPrice: low, HighPrice: high, Note: note}, true
}

func formatIntWithComma(n int64) string {
	if n == math.MinInt64 {
		// MinInt64 は -n がオーバーフローするため再帰処理不可。定数で返す。
		return "-9,223,372,036,854,775,808"
	}
	if n < 0 {
		return "-" + formatIntWithComma(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatIntWithComma(n/1000) + fmt.Sprintf(",%03d", n%1000)
}
