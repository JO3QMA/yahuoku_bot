package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_envDefaults(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", " tok ")
	t.Setenv("GEMINI_API_KEY", "k")
	t.Setenv("API_ENDPOINT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("RQLITE_URL", "")
	t.Setenv("CHECK_INTERVAL_MINUTES", "notint")
	t.Setenv("POLL_DELAY_MS", "abc")

	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "missing.yaml"))
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
	if cfg.DBPath != "data/watch.db" {
		t.Fatalf("DBPath=%q", cfg.DBPath)
	}
	if cfg.CheckIntervalMinutes != 5 || cfg.PollDelayMs != 2000 {
		t.Fatalf("intervals: %d %d", cfg.CheckIntervalMinutes, cfg.PollDelayMs)
	}
	if cfg.GeminiMaxImages != 3 || cfg.GeminiMaxSearchCalls != 3 {
		t.Fatalf("gemini limits: %d %d", cfg.GeminiMaxImages, cfg.GeminiMaxSearchCalls)
	}
}

func TestLoad_yamlAllowed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(path, []byte("allowed:\n  guilds: [g1]\n  channels: [c1]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AllowedGuilds) != 1 || cfg.AllowedGuilds[0] != "g1" {
		t.Fatalf("guilds %+v", cfg.AllowedGuilds)
	}
	if len(cfg.AllowedChannels) != 1 || cfg.AllowedChannels[0] != "c1" {
		t.Fatalf("channels %+v", cfg.AllowedChannels)
	}
}

func TestLoad_readError(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error reading directory as file")
	}
}

func TestLoad_badYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("allowed: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoad_invalidEnvIntUsesDefaults(t *testing.T) {
	t.Setenv("CHECK_INTERVAL_MINUTES", "not-a-number")
	t.Setenv("POLL_DELAY_MS", "xyz")
	t.Cleanup(func() {
		_ = os.Unsetenv("CHECK_INTERVAL_MINUTES")
		_ = os.Unsetenv("POLL_DELAY_MS")
	})
	_, err := Load(filepath.Join(t.TempDir(), "none.yaml"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoad_validEnvInts(t *testing.T) {
	t.Setenv("CHECK_INTERVAL_MINUTES", "17")
	t.Setenv("POLL_DELAY_MS", "42")
	t.Cleanup(func() {
		_ = os.Unsetenv("CHECK_INTERVAL_MINUTES")
		_ = os.Unsetenv("POLL_DELAY_MS")
	})
	cfg, err := Load(filepath.Join(t.TempDir(), "none.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CheckIntervalMinutes != 17 || cfg.PollDelayMs != 42 {
		t.Fatalf("got %d %d", cfg.CheckIntervalMinutes, cfg.PollDelayMs)
	}
}
