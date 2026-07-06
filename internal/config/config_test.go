package config

import (
	"os"
	"testing"
)

func TestLoad_envDefaults(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", " tok ")
	t.Setenv("GEMINI_API_KEY", "k")
	t.Setenv("API_ENDPOINT", "")
	t.Setenv("RQLITE_URL", "")
	t.Setenv("ALLOWED_GUILDS", "")
	t.Setenv("ALLOWED_CHANNELS", "")
	t.Setenv("CHECK_INTERVAL_MINUTES", "notint")
	t.Setenv("POLL_DELAY_MS", "abc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DiscordToken != "tok" {
		t.Fatalf("DiscordToken=%q", cfg.DiscordToken)
	}
	if cfg.GeminiAPIKey != "k" {
		t.Fatalf("GeminiAPIKey")
	}
	if cfg.APIEndpoint != "http://localhost:8080" {
		t.Fatalf("APIEndpoint=%q", cfg.APIEndpoint)
	}
	if cfg.RqliteURL != "http://localhost:4001" {
		t.Fatalf("RqliteURL=%q", cfg.RqliteURL)
	}
	if cfg.CheckIntervalMinutes != 5 || cfg.PollDelayMs != 2000 {
		t.Fatalf("intervals: %d %d", cfg.CheckIntervalMinutes, cfg.PollDelayMs)
	}
	if cfg.GeminiMaxImages != 3 || cfg.GeminiMaxSearchCalls != 3 || cfg.GeminiPipelineTimeoutSec != 45 {
		t.Fatalf("gemini limits: %d %d timeout %d", cfg.GeminiMaxImages, cfg.GeminiMaxSearchCalls, cfg.GeminiPipelineTimeoutSec)
	}
	if cfg.MarketEstimateMinSamples != 5 || cfg.MarketEstimateLookbackDays != 90 {
		t.Fatalf("market estimate: %d %d", cfg.MarketEstimateMinSamples, cfg.MarketEstimateLookbackDays)
	}
	if cfg.MarketEstimateTimeoutSec != 20 || cfg.HandlerMarketTimeoutSec != 25 {
		t.Fatalf("market timeouts: %d %d", cfg.MarketEstimateTimeoutSec, cfg.HandlerMarketTimeoutSec)
	}
}

func TestLoad_allowedCSV(t *testing.T) {
	t.Setenv("ALLOWED_GUILDS", "g1, g2")
	t.Setenv("ALLOWED_CHANNELS", "c1")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AllowedGuilds) != 2 || cfg.AllowedGuilds[0] != "g1" || cfg.AllowedGuilds[1] != "g2" {
		t.Fatalf("guilds %+v", cfg.AllowedGuilds)
	}
	if len(cfg.AllowedChannels) != 1 || cfg.AllowedChannels[0] != "c1" {
		t.Fatalf("channels %+v", cfg.AllowedChannels)
	}
}

func TestLoad_invalidEnvIntUsesDefaults(t *testing.T) {
	t.Setenv("CHECK_INTERVAL_MINUTES", "not-a-number")
	t.Setenv("POLL_DELAY_MS", "xyz")
	t.Cleanup(func() {
		_ = os.Unsetenv("CHECK_INTERVAL_MINUTES")
		_ = os.Unsetenv("POLL_DELAY_MS")
	})
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CheckIntervalMinutes != 5 || cfg.PollDelayMs != 2000 {
		t.Fatalf("got %d %d", cfg.CheckIntervalMinutes, cfg.PollDelayMs)
	}
}

func TestLoad_validEnvInts(t *testing.T) {
	t.Setenv("CHECK_INTERVAL_MINUTES", "17")
	t.Setenv("POLL_DELAY_MS", "42")
	t.Cleanup(func() {
		_ = os.Unsetenv("CHECK_INTERVAL_MINUTES")
		_ = os.Unsetenv("POLL_DELAY_MS")
	})
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CheckIntervalMinutes != 17 || cfg.PollDelayMs != 42 {
		t.Fatalf("got %d %d", cfg.CheckIntervalMinutes, cfg.PollDelayMs)
	}
}

func TestLoad_geminiPipelineTimeout(t *testing.T) {
	t.Setenv("GEMINI_PIPELINE_TIMEOUT_SEC", "90")
	t.Cleanup(func() { _ = os.Unsetenv("GEMINI_PIPELINE_TIMEOUT_SEC") })
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GeminiPipelineTimeoutSec != 90 {
		t.Fatalf("got %d", cfg.GeminiPipelineTimeoutSec)
	}
}

func TestLoad_handlerTimeoutNotLessThanMarket(t *testing.T) {
	t.Setenv("MARKET_ESTIMATE_TIMEOUT_SEC", "30")
	t.Setenv("HANDLER_MARKET_TIMEOUT_SEC", "10")
	t.Cleanup(func() {
		_ = os.Unsetenv("MARKET_ESTIMATE_TIMEOUT_SEC")
		_ = os.Unsetenv("HANDLER_MARKET_TIMEOUT_SEC")
	})
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MarketEstimateTimeoutSec != 30 || cfg.HandlerMarketTimeoutSec != 30 {
		t.Fatalf("got market=%d handler=%d", cfg.MarketEstimateTimeoutSec, cfg.HandlerMarketTimeoutSec)
	}
}
