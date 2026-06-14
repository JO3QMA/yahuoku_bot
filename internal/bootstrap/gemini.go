package bootstrap

import (
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
)

// GeminiOptions はアプリ設定から多段推論パイプラインの Options を構築する。
func GeminiOptions(cfg *config.Config) gemini.Options {
	if cfg == nil {
		return gemini.Options{}.Normalize()
	}
	return gemini.Options{
		APIKey:             cfg.GeminiAPIKey,
		FastModel:          cfg.GeminiModel,
		VisionModel:        cfg.GeminiModelVision,
		AgentModel:         cfg.GeminiModelAgent,
		MaxImages:          cfg.GeminiMaxImages,
		MaxSearchCalls:     cfg.GeminiMaxSearchCalls,
		PipelineTimeoutSec: cfg.GeminiPipelineTimeoutSec,
	}.Normalize()
}
