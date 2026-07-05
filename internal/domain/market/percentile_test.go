package market

import "testing"

func TestPercentile_range(t *testing.T) {
	prices := []int64{5000, 8000, 10000, 12000, 50000}
	est, ok := FromPrices(prices, "test")
	if !ok {
		t.Fatal("expected ok")
	}
	if est.LowPrice != 8000 {
		t.Fatalf("low=%d want 8000", est.LowPrice)
	}
	if est.HighPrice != 12000 {
		t.Fatalf("high=%d want 12000", est.HighPrice)
	}
}

func TestFromPrices_empty(t *testing.T) {
	_, ok := FromPrices(nil, "x")
	if ok {
		t.Fatal("expected false")
	}
}

func TestFromPrices_single(t *testing.T) {
	est, ok := FromPrices([]int64{9000}, "one")
	if !ok || est.LowPrice != 9000 || est.HighPrice != 9000 {
		t.Fatalf("got %+v ok=%v", est, ok)
	}
}

func TestFromPrices_allSame(t *testing.T) {
	est, ok := FromPrices([]int64{5000, 5000, 5000, 5000, 5000}, "same")
	if !ok || est.LowPrice != 5000 || est.HighPrice != 5000 {
		t.Fatalf("got %+v ok=%v", est, ok)
	}
}

func TestFromPrices_twoElements(t *testing.T) {
	est, ok := FromPrices([]int64{8000, 12000}, "two")
	if !ok || est.LowPrice != 9000 || est.HighPrice != 11000 {
		t.Fatalf("got %+v ok=%v", est, ok)
	}
}
