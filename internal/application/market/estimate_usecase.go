package market

import (
	"context"
	"fmt"
	"log"

	domainmarket "jo3qma.com/yahoo_auctions_bot/internal/domain/market"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// Config は MarketEstimate 算出の設定。
type Config struct {
	MinSamples   int
	LookbackDays int
}

// Normalize は未設定値を既定値で埋める。
func (c Config) Normalize() Config {
	if c.MinSamples <= 0 {
		c.MinSamples = 5
	}
	if c.LookbackDays <= 0 {
		c.LookbackDays = 90
	}
	return c
}

// SoldComparableSearcher は落札済み Comparable の価格一覧を返す。
type SoldComparableSearcher interface {
	SearchSoldPrices(ctx context.Context, category product.Category, identityKey, identityValue string, lookbackDays int) ([]int64, error)
}

// WebMarketEstimator は Web 検索ベースで MarketEstimate を返す。
type WebMarketEstimator interface {
	Estimate(ctx context.Context, title, description string, p *product.Product, identityMissing bool) (*domainmarket.MarketEstimate, error)
}

// EstimateUsecase は MarketEstimate を算出する。
type EstimateUsecase struct {
	sold SoldComparableSearcher
	web  WebMarketEstimator
	cfg  Config
}

// NewEstimateUsecase は EstimateUsecase を生成する。
func NewEstimateUsecase(sold SoldComparableSearcher, web WebMarketEstimator, cfg Config) *EstimateUsecase {
	return &EstimateUsecase{sold: sold, web: web, cfg: cfg.Normalize()}
}

// Execute は Product から MarketEstimate を算出する。取得不可のときは nil, nil。
func (u *EstimateUsecase) Execute(ctx context.Context, title, description string, p *product.Product) (*domainmarket.MarketEstimate, error) {
	if u == nil || p == nil {
		return nil, nil
	}
	cfg := u.cfg

	key, identityValue, hasIdentity := domainmarket.IdentityValue(p)
	if !hasIdentity {
		return u.webEstimate(ctx, title, description, p, true)
	}

	if u.sold != nil {
		prices, err := u.sold.SearchSoldPrices(ctx, p.Category, key, identityValue, cfg.LookbackDays)
		if err != nil {
			log.Printf("[market] search sold comparables: %v", err)
			return u.webEstimate(ctx, title, description, p, false)
		}
		if len(prices) >= cfg.MinSamples {
			note := fmt.Sprintf("ヤフオク落札 %d 件・直近%d日", len(prices), cfg.LookbackDays)
			est, ok := domainmarket.FromPrices(prices, note)
			if ok {
				return est, nil
			}
		}
	}

	return u.webEstimate(ctx, title, description, p, false)
}

func (u *EstimateUsecase) webEstimate(ctx context.Context, title, description string, p *product.Product, identityMissing bool) (*domainmarket.MarketEstimate, error) {
	if u.web == nil {
		return nil, nil
	}
	return u.web.Estimate(ctx, title, description, p, identityMissing)
}
