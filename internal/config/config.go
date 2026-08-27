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
	OpenAIAPIKey             string
	OpenAIBaseURL            string // OpenAI 互換 API のベース URL (default: https://api.openai.com/v1)
	OpenAIModel              string // Stage1/4 用。空の場合は gpt-4o-mini
	OpenAIModelVision        string // Stage2 用。空の場合は gpt-4o
	OpenAIModelAgent         string // Stage3 用。空の場合は gpt-4o
	OpenAIMaxImages          int    // 推論に使う最大画像数 (default: 3)
	OpenAIMaxSearchCalls     int    // 1商品あたりの最大検索回数 (default: 3)
	OpenAIPipelineTimeoutSec int    // Extraction のタイムアウト秒 (default: 45)
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
		OpenAIAPIKey:             strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIBaseURL:            strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		OpenAIModel:              strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		OpenAIModelVision:        strings.TrimSpace(os.Getenv("OPENAI_MODEL_VISION")),
		OpenAIModelAgent:         strings.TrimSpace(os.Getenv("OPENAI_MODEL_AGENT")),
		OpenAIMaxImages:          getEnvInt("OPENAI_MAX_IMAGES", 3),
		OpenAIMaxSearchCalls:     getEnvInt("OPENAI_MAX_SEARCH_CALLS", 3),
		OpenAIPipelineTimeoutSec: getEnvInt("OPENAI_PIPELINE_TIMEOUT_SEC", 45),
		RqliteURL:                strings.TrimSpace(os.Getenv("RQLITE_URL")),
		AllowedGuilds:            getEnvCSV("ALLOWED_GUILDS"),
		AllowedChannels:          getEnvCSV("ALLOWED_CHANNELS"),
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
