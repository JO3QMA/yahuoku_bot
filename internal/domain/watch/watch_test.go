package watch

import "testing"

func TestWatch_zero(t *testing.T) {
	var w Watch
	if w.ID != 0 || w.AuctionID != "" {
		t.Fatal()
	}
}
