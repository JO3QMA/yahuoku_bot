package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config は環境変数から読み込む実行時設定。
type Config struct {
	DiscordToken             string
	GeminiAPIKey             string
	GeminiModel              string // Stage1/4 用。空の場合は gemini-2.5-flash-lite
	GeminiModelVision        string // Stage2 用。空の場合は gemini-2.5-flash
	GeminiModelAgent         string // Stage3 用。空の場合は gemini-2.5-flash
	GeminiMaxImages          int    // 推論に使う最大画像数 (default: 3)
	GeminiMaxSearchCalls     int    // 1商品あたりの最大検索回数 (default: 3)
	GeminiPipelineTimeoutSec int    // 多段推論パイプラインのタイムアウト秒 (default: 45)
	APIEndpoint              string
	AllowedGuilds            []string // 空 = 全サーバー許可
	AllowedChannels          []string // 空 = 全チャンネル許可
	RqliteURL                string   // rqlite のベース URL (default: http://localhost:4001)
	CheckIntervalMinutes     int      // ポーリング間隔（分） (default: 5)
	PollDelayMs              int      // ポーリング時の1件あたりのディレイ（ms） (default: 2000)
}

// Load は環境変数から Config を返す。
// 環境変数は direnv 等で .env を読み込んだ状態で起動すること。
func Load() (*Config, error) {
	cfg := &Config{
		DiscordToken:             strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		GeminiAPIKey:             strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:              strings.TrimSpace(os.Getenv("GEMINI_MODEL")),
		GeminiModelVision:        strings.TrimSpace(os.Getenv("GEMINI_MODEL_VISION")),
		GeminiModelAgent:         strings.TrimSpace(os.Getenv("GEMINI_MODEL_AGENT")),
		GeminiMaxImages:          getEnvInt("GEMINI_MAX_IMAGES", 3),
		GeminiMaxSearchCalls:     getEnvInt("GEMINI_MAX_SEARCH_CALLS", 3),
		GeminiPipelineTimeoutSec: getEnvInt("GEMINI_PIPELINE_TIMEOUT_SEC", 45),
		APIEndpoint:              strings.TrimSpace(os.Getenv("API_ENDPOINT")),
		RqliteURL:                strings.TrimSpace(os.Getenv("RQLITE_URL")),
		AllowedGuilds:            getEnvCSV("ALLOWED_GUILDS"),
		AllowedChannels:          getEnvCSV("ALLOWED_CHANNELS"),
	}

	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "http://localhost:8080"
	}
	if cfg.RqliteURL == "" {
		cfg.RqliteURL = "http://localhost:4001"
	}
	cfg.CheckIntervalMinutes = getEnvInt("CHECK_INTERVAL_MINUTES", 5)
	cfg.PollDelayMs = getEnvInt("POLL_DELAY_MS", 2000)

	return cfg, nil
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("[Config] Warning: invalid %s value %q, using default %d", key, v, fallback)
		return fallback
	}
	return n
}

func getEnvCSV(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
