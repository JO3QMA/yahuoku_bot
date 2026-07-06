package market

import (
	"math"
	"testing"
)

func TestMarketEstimate_DisplayValue_nil(t *testing.T) {
	var m *MarketEstimate
	if got := m.DisplayValue(); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestMarketEstimate_DisplayValue(t *testing.T) {
	m := &MarketEstimate{LowPrice: 10000, HighPrice: 15000, Note: "テスト"}
	got := m.DisplayValue()
	want := "¥10,000〜¥15,000（テスト）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMarketEstimate_DisplayValue_negative(t *testing.T) {
	m := &MarketEstimate{LowPrice: -1200, HighPrice: 500, Note: "x"}
	got := m.DisplayValue()
	if got == "" {
		t.Fatal("empty")
	}
}

func TestMarketEstimate_DisplayValue_minInt64(t *testing.T) {
	m := &MarketEstimate{LowPrice: math.MinInt64, HighPrice: 1, Note: "edge"}
	got := m.DisplayValue()
	if got == "" {
		t.Fatal("empty")
	}
}
