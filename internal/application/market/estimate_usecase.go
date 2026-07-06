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
	if c.MinSamples > 100 {
		c.MinSamples = 100
	}
	if c.LookbackDays <= 0 {
		c.LookbackDays = 90
	}
	if c.LookbackDays > 365 {
		c.LookbackDays = 365
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
	if u == nil {
		return nil, fmt.Errorf("EstimateUsecase.Execute: nil receiver")
	}
	if p == nil {
		return nil, fmt.Errorf("EstimateUsecase.Execute: nil product")
	}
	cfg := u.cfg

	key, identityValue, hasIdentity := domainmarket.IdentityValue(p)
	if !hasIdentity {
		return u.webEstimate(ctx, title, description, p, true)
	}

	// Sold 検索失敗・サンプル不足時はエラーを返さず Web 推定へフォールバックする（グレースフルデグラデーション）。
	// 落札データ由来か Web 推定由来かは MarketEstimate.Note で区別する。
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
			log.Printf("[market] FromPrices returned false despite %d samples", len(prices))
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
