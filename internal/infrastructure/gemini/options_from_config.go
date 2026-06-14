package gemini

import "jo3qma.com/yahoo_auctions_bot/internal/config"

// OptionsFromConfig はアプリ設定からパイプライン Options を構築する。
func OptionsFromConfig(cfg *config.Config) Options {
	if cfg == nil {
		return Options{}.Normalize()
	}
	return Options{
		APIKey:         cfg.GeminiAPIKey,
		FastModel:      cfg.GeminiModel,
		VisionModel:    cfg.GeminiModelVision,
		AgentModel:     cfg.GeminiModelAgent,
		MaxImages:      cfg.GeminiMaxImages,
		MaxSearchCalls: cfg.GeminiMaxSearchCalls,
	}.Normalize()
}
