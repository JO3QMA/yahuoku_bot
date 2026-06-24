package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
	infrarqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/rqlite"
	"jo3qma.com/yahoo_auctions_bot/internal/presentation/discord"
)

type discordRunner interface {
	Run(context.Context) error
}

// botDeps は run の依存注入用（テストで差し替え）。
type botDeps struct {
	LoadConfig      func() (*config.Config, error)
	NewGeminiClient func(cfg *config.Config) (appauction.Extractor, error)
	OpenRqlite      func(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error)
	NewWatchRepo    func(*infrarqlite.Client) watch.Repository
	NewDiscordBot   func(token string, pu *appauction.PreviewUsecase, af *discord.AllowedFilter, wu *appwatch.WatchUsecase, ac infraauction.Client, repo watch.Repository, cfg discord.BotConfig) (discordRunner, error)
}

// runWithSignalHook が nil でないとき runWithSignal を置き換える（本パッケージのテスト専用）。
var runWithSignalHook func(parent context.Context, deps *botDeps) error

func runWithSignal(parent context.Context, deps *botDeps) error {
	if runWithSignalHook != nil {
		return runWithSignalHook(parent, deps)
	}
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
	if d.NewGeminiClient == nil {
		d.NewGeminiClient = func(cfg *config.Config) (appauction.Extractor, error) {
			opts := gemini.NewOptions(
				cfg.GeminiModel, cfg.GeminiModelVision, cfg.GeminiModelAgent,
				cfg.GeminiMaxImages, cfg.GeminiMaxSearchCalls, cfg.GeminiPipelineTimeoutSec,
			)
			return gemini.NewClient(cfg.GeminiAPIKey, opts)
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
	if d.NewDiscordBot == nil {
		d.NewDiscordBot = defaultNewDiscordBot
	}
}

func defaultNewDiscordBot(token string, pu *appauction.PreviewUsecase, af *discord.AllowedFilter, wu *appwatch.WatchUsecase, ac infraauction.Client, repo watch.Repository, cfg discord.BotConfig) (discordRunner, error) {
	return discord.NewBot(token, pu, af, wu, ac, repo, cfg)
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
	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required")
	}

	auctionClient := infraauction.NewClient(cfg.APIEndpoint, nil)
	geminiClient, err := deps.NewGeminiClient(cfg)
	if err != nil {
		return fmt.Errorf("gemini client: %w", err)
	}

	rqliteClient, err := deps.OpenRqlite(ctx, cfg.RqliteURL)
	if err != nil {
		return fmt.Errorf("rqlite open: %w", err)
	}
	defer func() { _ = rqliteClient.Close() }()
	watchRepo := deps.NewWatchRepo(rqliteClient)

	previewUsecase := appauction.NewPreviewUsecase(auctionClient, geminiClient)
	watchUsecase := appwatch.NewWatchUsecase(watchRepo)

	allowedFilter := discord.NewAllowedFilter(cfg.AllowedGuilds, cfg.AllowedChannels)
	discordToken := cfg.DiscordToken
	if discordToken != "" && !strings.HasPrefix(discordToken, "Bot ") {
		discordToken = "Bot " + discordToken
	}

	bot, err := deps.NewDiscordBot(
		discordToken,
		previewUsecase,
		allowedFilter,
		watchUsecase,
		auctionClient,
		watchRepo,
		discord.BotConfig{
			CheckIntervalMinutes: cfg.CheckIntervalMinutes,
			PollDelayMs:          cfg.PollDelayMs,
		},
	)
	if err != nil {
		return fmt.Errorf("bot init: %w", err)
	}

	if err := bot.Run(ctx); err != nil && err != context.Canceled {
		log.Printf("[yahoo_auctions_bot] bot run: %v", err)
	}
	return nil
}
