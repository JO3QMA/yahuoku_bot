package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Allowed はBotが反応するサーバー・チャンネルのフィルタ設定。
// 空の場合は全サーバー・全チャンネルを許可する。
type Allowed struct {
	Guilds   []string `yaml:"guilds"`   // 空 = 全サーバー許可
	Channels []string `yaml:"channels"` // 空 = 全チャンネル許可
}

// YAMLConfig はconfig.yamlの構造。
type YAMLConfig struct {
	Allowed Allowed `yaml:"allowed"`
}

// Config は環境変数とYAMLを統合した実行時設定。
type Config struct {
	DiscordToken    string
	GeminiAPIKey    string
	GeminiModel     string // 空の場合は gemini-1.5-flash が使われる
	APIEndpoint     string
	AllowedGuilds   []string
	AllowedChannels []string

	DBPath               string // SQLiteデータベースパス (default: "data/watch.db")。RqliteURL が空のときのみ使用。
	RqliteURL            string // rqlite のベース URL (例: http://rqlite:4001)。設定時はこちらを優先し DB は rqlite に接続する。
	CheckIntervalMinutes int    // ポーリング間隔（分） (default: 5)
	PollDelayMs          int    // ポーリング時の1件あたりのディレイ（ms） (default: 2000)
}

// Load は環境変数とconfigPathのYAMLを読み込み、Configを返す。
// 環境変数は direnv 等で .env を読み込んだ状態で起動すること。
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		DiscordToken: strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		GeminiAPIKey: strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:  strings.TrimSpace(os.Getenv("GEMINI_MODEL")),
		APIEndpoint:  strings.TrimSpace(os.Getenv("API_ENDPOINT")),
		DBPath:    strings.TrimSpace(os.Getenv("DB_PATH")),
		RqliteURL: strings.TrimSpace(os.Getenv("RQLITE_URL")),
	}

	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "http://localhost:8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "data/watch.db"
	}
	cfg.CheckIntervalMinutes = getEnvInt("CHECK_INTERVAL_MINUTES", 5)
	cfg.PollDelayMs = getEnvInt("POLL_DELAY_MS", 2000)

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if len(data) > 0 {
			var yc YAMLConfig
			if err := yaml.Unmarshal(data, &yc); err != nil {
				return nil, fmt.Errorf("parse config yaml: %w", err)
			}
			cfg.AllowedGuilds = yc.Allowed.Guilds
			cfg.AllowedChannels = yc.Allowed.Channels
		}
	}

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
