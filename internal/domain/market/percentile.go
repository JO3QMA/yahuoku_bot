package market

// percentile はソート済み配列の p パーセンタイル（0〜1）を返す。
func percentile(sorted []int64, p float64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := p * float64(n-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= n {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return int64(float64(sorted[lo])*(1-frac) + float64(sorted[hi])*frac)
}

func sortInt64s(a []int64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
