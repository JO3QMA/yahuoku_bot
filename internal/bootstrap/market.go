package bootstrap

import (
	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
)

// MarketEstimateUsecase は設定から MarketEstimate 算出ユースケースを構築する。
func MarketEstimateUsecase(cfg *config.Config) (*appmarket.EstimateUsecase, error) {
	if cfg == nil {
		return nil, nil
	}
	sold := infraauction.NewSoldComparableSearcher(cfg.APIEndpoint, nil)
	web, err := gemini.NewMarketEstimator(cfg.GeminiAPIKey, cfg.GeminiModelAgent)
	if err != nil {
		return nil, err
	}
	return appmarket.NewEstimateUsecase(sold, web, appmarket.Config{
		MinSamples:   cfg.MarketEstimateMinSamples,
		LookbackDays: cfg.MarketEstimateLookbackDays,
	}), nil
}
