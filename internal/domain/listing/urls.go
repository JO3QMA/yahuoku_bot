package listing

import "regexp"

type urlPattern struct {
	market Market
	re     *regexp.Regexp
}

var listingURLPatterns = []urlPattern{
	{MarketYahooAuction, regexp.MustCompile(`(?:^|[^a-zA-Z0-9])auctions?\.yahoo\.co\.jp/[^/]+/auction/([a-zA-Z0-9]{8,11})`)},
	{MarketYahooFlea, regexp.MustCompile(`(?:^|[^a-zA-Z0-9])paypayfleamarket\.yahoo\.co\.jp/item/([a-zA-Z0-9]+)`)},
	{MarketMercari, regexp.MustCompile(`(?:^|[^a-zA-Z0-9])jp\.mercari\.com/item/([a-zA-Z0-9]+)`)},
}

// ParseRefs はテキスト中の対応 URL から Listing 参照を抽出する（Market 定義順、重複は呼び出し側で除去）。
func ParseRefs(text string) []Ref {
	var out []Ref
	for _, p := range listingURLPatterns {
		for _, m := range p.re.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}
			out = append(out, Ref{Market: p.market, ListingID: m[1]})
		}
	}
	return out
}

// ParseRefFromURL は単一 URL 文字列から最初にマッチした Listing 参照を返す。
func ParseRefFromURL(raw string) (Ref, bool) {
	for _, ref := range ParseRefs(raw) {
		return ref, true
	}
	return Ref{}, false
}
