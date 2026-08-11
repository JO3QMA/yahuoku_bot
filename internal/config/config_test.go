package config

import (
	"os"
	"testing"
)

func TestLoad_envDefaults(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", " tok ")
	t.Setenv("OPENAI_API_KEY", "k")
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
	if cfg.OpenAIAPIKey != "k" {
		t.Fatalf("OpenAIAPIKey")
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
	if cfg.OpenAIMaxImages != 3 || cfg.OpenAIMaxSearchCalls != 3 || cfg.OpenAIPipelineTimeoutSec != 45 {
		t.Fatalf("openai limits: %d %d timeout %d", cfg.OpenAIMaxImages, cfg.OpenAIMaxSearchCalls, cfg.OpenAIPipelineTimeoutSec)
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

func TestLoad_openaiPipelineTimeout(t *testing.T) {
	t.Setenv("OPENAI_PIPELINE_TIMEOUT_SEC", "90")
	t.Cleanup(func() { _ = os.Unsetenv("OPENAI_PIPELINE_TIMEOUT_SEC") })
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIPipelineTimeoutSec != 90 {
		t.Fatalf("got %d", cfg.OpenAIPipelineTimeoutSec)
	}
}

func TestLoad_openaiBaseURL(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://custom.example.com/v1")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIBaseURL != "https://custom.example.com/v1" {
		t.Fatalf("got %q", cfg.OpenAIBaseURL)
	}
}
