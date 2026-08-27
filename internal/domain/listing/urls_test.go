package listing

import "testing"

func TestParseRefs(t *testing.T) {
	text := `ヤフオク https://page.auctions.yahoo.co.jp/jp/auction/o123456789
フリマ https://paypayfleamarket.yahoo.co.jp/item/z668531248
メルカリ https://jp.mercari.com/item/m56797713000`
	refs := ParseRefs(text)
	if len(refs) != 3 {
		t.Fatalf("got %d refs: %#v", len(refs), refs)
	}
	if refs[0] != (Ref{Market: MarketYahooAuction, ListingID: "o123456789"}) {
		t.Fatalf("yahoo: %#v", refs[0])
	}
	if refs[1] != (Ref{Market: MarketYahooFlea, ListingID: "z668531248"}) {
		t.Fatalf("flea: %#v", refs[1])
	}
	if refs[2] != (Ref{Market: MarketMercari, ListingID: "m56797713000"}) {
		t.Fatalf("mercari: %#v", refs[2])
	}
}

func TestParseRefFromURL(t *testing.T) {
	ref, ok := ParseRefFromURL("https://jp.mercari.com/item/m123")
	if !ok || ref.Market != MarketMercari || ref.ListingID != "m123" {
		t.Fatalf("got %#v ok=%v", ref, ok)
	}
	_, ok = ParseRefFromURL("https://example.com/")
	if ok {
		t.Fatal("expected no match")
	}
}
