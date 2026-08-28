package listing

import "time"

// Market は Listing が属する C2C サービス。
type Market string

const (
	MarketYahooAuction Market = "yahoo_auction"
	MarketYahooFlea    Market = "yahoo_flea"
	MarketMercari      Market = "mercari"
)

// Valid は既知の Market かを返す。
func (m Market) Valid() bool {
	switch m {
	case MarketYahooAuction, MarketYahooFlea, MarketMercari:
		return true
	default:
		return false
	}
}

// SaleType は Listing の販売形式。
type SaleType string

const (
	SaleTypeAuction    SaleType = "auction"
	SaleTypeFixedPrice SaleType = "fixed_price"
)

// Ref は market と listingID の組で Listing を識別する。
type Ref struct {
	Market    Market
	ListingID string
}

// Data は sansai から取得した Listing 属性。
type Data struct {
	Ref         Ref
	Title       string
	URL         string
	Price       int64
	Status      string
	Description string
	ImageURLs   []string
	SaleType    SaleType
	EndTime     *time.Time
	IsActive    bool
}

// IsAuction は入札・終了時刻を持つ販売形式かを返す。
func (d *Data) IsAuction() bool {
	return d != nil && d.SaleType == SaleTypeAuction
}
