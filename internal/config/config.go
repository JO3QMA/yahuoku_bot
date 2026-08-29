package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// Config は環境変数から読み込む実行時設定。
type Config struct {
	DiscordToken string
	OpenAIAPIKey string
	OpenAI       openai.Options
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
		DiscordToken: strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		OpenAIAPIKey: strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAI: openai.Options{
			BaseURL:            strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
			FastModel:          strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
			VisionModel:        strings.TrimSpace(os.Getenv("OPENAI_MODEL_VISION")),
			AgentModel:         strings.TrimSpace(os.Getenv("OPENAI_MODEL_AGENT")),
			MaxImages:          getEnvInt("OPENAI_MAX_IMAGES", 0),
			MaxSearchCalls:     getEnvInt("OPENAI_MAX_SEARCH_CALLS", 0),
			PipelineTimeoutSec: getEnvInt("OPENAI_PIPELINE_TIMEOUT_SEC", 0),
		}.Normalize(),
		RqliteURL:       strings.TrimSpace(os.Getenv("RQLITE_URL")),
		AllowedGuilds:   getEnvCSV("ALLOWED_GUILDS"),
		AllowedChannels: getEnvCSV("ALLOWED_CHANNELS"),
	}

	if cfg.RqliteURL == "" {
		cfg.RqliteURL = "http://localhost:4001"
	}
	cfg.CheckIntervalMinutes = getEnvInt("CHECK_INTERVAL_MINUTES", 5)
	cfg.PollDelayMs = getEnvInt("POLL_DELAY_MS", 2000)

	return cfg, nil
}

// NewOpenAIClient は Config から OpenAI Extraction クライアントを生成する。
func NewOpenAIClient(cfg *Config) (openai.Client, error) {
	return openai.NewClient(cfg.OpenAIAPIKey, cfg.OpenAI)
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
