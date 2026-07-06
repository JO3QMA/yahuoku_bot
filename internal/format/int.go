package format

import (
	"fmt"
	"math"
)

// IntWithComma は整数を3桁カンマ区切りの文字列にする。
func IntWithComma(n int64) string {
	if n == math.MinInt64 {
		// MinInt64 は -n がオーバーフローするため再帰処理不可。定数で返す。
		return "-9,223,372,036,854,775,808"
	}
	if n < 0 {
		return "-" + IntWithComma(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return IntWithComma(n/1000) + fmt.Sprintf(",%03d", n%1000)
}
