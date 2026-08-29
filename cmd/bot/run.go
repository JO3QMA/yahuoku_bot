package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
	infrarqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/rqlite"
	"jo3qma.com/yahoo_auctions_bot/internal/presentation/discord"
)

// botDeps は run の依存注入用（テストで差し替え）。
type botDeps struct {
	LoadConfig      func() (*config.Config, error)
	NewOpenAIClient func(cfg *config.Config) (openai.Client, error)
	OpenRqlite      func(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error)
	NewWatchRepo    func(*infrarqlite.Client) watch.Repository
}

func runWithSignal(parent context.Context, deps *botDeps) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-sigCh:
			cancel()
		}
	}()

	return run(ctx, deps)
}

func mergeBotDeps(d *botDeps) {
	if d.LoadConfig == nil {
		d.LoadConfig = config.Load
	}
	if d.NewOpenAIClient == nil {
		d.NewOpenAIClient = func(cfg *config.Config) (openai.Client, error) {
			opts := openai.NewOptions(
				cfg.OpenAIBaseURL, cfg.OpenAIModel, cfg.OpenAIModelVision, cfg.OpenAIModelAgent,
				cfg.OpenAIMaxImages, cfg.OpenAIMaxSearchCalls, cfg.OpenAIPipelineTimeoutSec,
			)
			return openai.NewClient(cfg.OpenAIAPIKey, opts)
		}
	}
	if d.OpenRqlite == nil {
		d.OpenRqlite = infrarqlite.Open
	}
	if d.NewWatchRepo == nil {
		d.NewWatchRepo = func(c *infrarqlite.Client) watch.Repository {
			return infrarqlite.NewWatchRepository(c)
		}
	}
}

func discordBotToken(raw string) string {
	if raw != "" && !strings.HasPrefix(raw, "Bot ") {
		return "Bot " + raw
	}
	return raw
}

func run(ctx context.Context, deps *botDeps) error {
	if deps == nil {
		deps = &botDeps{}
	}
	mergeBotDeps(deps)

	cfg, err := deps.LoadConfig()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	if cfg.DiscordToken == "" {
		return fmt.Errorf("DISCORD_TOKEN is required")
	}
	if cfg.OpenAIAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	openaiClient, err := deps.NewOpenAIClient(cfg)
	if err != nil {
		return fmt.Errorf("openai client: %w", err)
	}

	rqliteClient, err := deps.OpenRqlite(ctx, cfg.RqliteURL)
	if err != nil {
		return fmt.Errorf("rqlite open: %w", err)
	}
	defer func() { _ = rqliteClient.Close() }()
	watchRepo := deps.NewWatchRepo(rqliteClient)

	previewUsecase := applisting.NewPreviewUsecase(openaiClient)

	allowedFilter := discord.NewAllowedFilter(cfg.AllowedGuilds, cfg.AllowedChannels)

	bot, err := discord.NewBot(
		discordBotToken(cfg.DiscordToken),
		previewUsecase,
		allowedFilter,
		watchRepo,
		cfg.CheckIntervalMinutes,
		cfg.PollDelayMs,
	)
	if err != nil {
		return fmt.Errorf("bot init: %w", err)
	}

	if err := bot.Run(ctx); err != nil && err != context.Canceled {
		log.Printf("[yahoo_auctions_bot] bot run: %v", err)
	}
	return nil
}
