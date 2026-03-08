package main

import (
	"context"
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
	infrasqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/sqlite"
	"jo3qma.com/yahoo_auctions_bot/internal/presentation/discord"
)

func main() {
	configPath := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		configPath = p
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("[yahoo_auctions_bot] config load: %v", err)
	}

	if cfg.DiscordToken == "" {
		log.Fatal("[yahoo_auctions_bot] DISCORD_TOKEN is required")
	}
	if cfg.GeminiAPIKey == "" {
		log.Fatal("[yahoo_auctions_bot] GEMINI_API_KEY is required")
	}

	// Infrastructure
	auctionClient := infraauction.NewClient(cfg.APIEndpoint, nil)
	geminiClient, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Fatalf("[yahoo_auctions_bot] gemini client: %v", err)
	}

	var watchRepo watch.Repository
	if cfg.RqliteURL != "" {
		rqliteClient, err := infrarqlite.Open(context.Background(), cfg.RqliteURL)
		if err != nil {
			log.Fatalf("[yahoo_auctions_bot] rqlite open: %v", err)
		}
		defer func() { _ = rqliteClient.Close() }()
		watchRepo = infrarqlite.NewWatchRepository(rqliteClient)
	} else {
		db, err := infrasqlite.Open(cfg.DBPath)
		if err != nil {
			log.Fatalf("[yahoo_auctions_bot] sqlite open: %v", err)
		}
		defer func() { _ = db.Close() }()
		watchRepo = infrasqlite.NewWatchRepository(db)
	}

	// Application
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, geminiClient)
	watchUsecase := appwatch.NewWatchUsecase(watchRepo)

	// Presentation
	allowedFilter := discord.NewAllowedFilter(cfg.AllowedGuilds, cfg.AllowedChannels)
	discordToken := cfg.DiscordToken
	if discordToken != "" && !strings.HasPrefix(discordToken, "Bot ") {
		discordToken = "Bot " + discordToken
	}

	bot, err := discord.NewBot(
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
		log.Fatalf("[yahoo_auctions_bot] bot init: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := bot.Run(ctx); err != nil && err != context.Canceled {
		log.Printf("[yahoo_auctions_bot] bot run: %v", err)
	}
}
