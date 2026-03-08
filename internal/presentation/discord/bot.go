package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"

	"jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// Bot はarikawa Stateとハンドラーを保持し、Discord Botを起動する。
type Bot struct {
	state           *state.State
	handler         *Handler
	reactionHandler *ReactionHandler
	pollingWorker   *appwatch.PollingWorker
}

// BotConfig は監視機能に関するBot設定。
type BotConfig struct {
	CheckIntervalMinutes int
	PollDelayMs          int
}

// NewBot はBotを生成する。DIは呼び出し元で行う。
func NewBot(
	token string,
	previewUsecase *auction.PreviewUsecase,
	allowed *AllowedFilter,
	watchUsecase *appwatch.WatchUsecase,
	auctionClient infraauction.Client,
	watchRepo domainwatch.Repository,
	botCfg BotConfig,
) (*Bot, error) {
	s := state.NewWithIntents(token,
		gateway.IntentGuilds,
		gateway.IntentGuildMessages,
		gateway.IntentDirectMessages,
		gateway.IntentMessageContent,
		gateway.IntentGuildMessageReactions,
	)

	embed := NewEmbedBuilder(s)
	h := NewHandler(previewUsecase, embed, allowed)

	reactionHandler := NewReactionHandler(watchUsecase, auctionClient, s, s)

	threadNotifier := NewThreadNotifier(s, watchRepo)
	pollingWorker := appwatch.NewPollingWorker(watchRepo, auctionClient, threadNotifier, botCfg.CheckIntervalMinutes, botCfg.PollDelayMs)

	return &Bot{
		state:           s,
		handler:         h,
		reactionHandler: reactionHandler,
		pollingWorker:   pollingWorker,
	}, nil
}

// Run はBotを起動し、Gatewayに接続する。ブロッキング。
func (b *Bot) Run(ctx context.Context) error {
	b.state.AddHandler(b.handler.HandleMessageCreate)
	b.state.AddHandler(b.reactionHandler.HandleReactionAdd)
	b.state.AddHandler(b.reactionHandler.HandleReactionRemove)

	if err := b.state.Connect(ctx); err != nil {
		return err
	}

	log.Println("[yahoo_auctions_bot] Bot started")

	go b.pollingWorker.Start(ctx)

	<-ctx.Done()
	return ctx.Err()
}
