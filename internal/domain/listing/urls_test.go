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

func TestParseRefFromURL_rejectsHostSubstring(t *testing.T) {
	_, ok := ParseRefFromURL("https://eviljp.mercari.com/item/m123")
	if ok {
		t.Fatal("expected no match for host substring")
	}
}

func TestParseRefFromURL_sansaiCanonicalURLs(t *testing.T) {
	cases := []struct {
		url    string
		market Market
		id     string
	}{
		{"https://auctions.yahoo.co.jp/jp/auction/o123456789", MarketYahooAuction, "o123456789"},
		{"https://paypayfleamarket.yahoo.co.jp/item/z668531248", MarketYahooFlea, "z668531248"},
		{"https://jp.mercari.com/item/m56797713000", MarketMercari, "m56797713000"},
	}
	for _, tc := range cases {
		ref, ok := ParseRefFromURL(tc.url)
		if !ok || ref != (Ref{Market: tc.market, ListingID: tc.id}) {
			t.Fatalf("%s: got %#v ok=%v", tc.url, ref, ok)
		}
	}
}
