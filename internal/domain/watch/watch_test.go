package watch

import "testing"

func TestWatchItem_zero(t *testing.T) {
	var w WatchItem
	if w.ID != 0 || w.AuctionID != "" {
		t.Fatal()
	}
}
