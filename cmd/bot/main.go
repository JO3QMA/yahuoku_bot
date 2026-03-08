package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
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

	// Application
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, geminiClient)

	// Presentation
	allowedFilter := discord.NewAllowedFilter(cfg.AllowedGuilds, cfg.AllowedChannels)
	// arikawa の REST API は Authorization にトークンをそのまま使う。Discord は Bot 用に "Bot " プレフィックスを要求する。
	discordToken := cfg.DiscordToken
	if discordToken != "" && !strings.HasPrefix(discordToken, "Bot ") {
		discordToken = "Bot " + discordToken
	}
	bot, err := discord.NewBot(discordToken, previewUsecase, allowedFilter)
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
