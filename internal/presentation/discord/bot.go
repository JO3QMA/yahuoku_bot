package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// Bot はarikawa Stateとハンドラーを保持し、Discord Botを起動する。
type Bot struct {
	state           *state.State
	handler         *Handler
	reactionHandler *ReactionHandler
	pollingWorker   *appwatch.PollingWorker
}

// NewBot はBotを生成する。DIは呼び出し元で行う。
func NewBot(
	token string,
	previewUsecase *applisting.PreviewUsecase,
	allowed *AllowedFilter,
	watchRepo domainwatch.Repository,
	checkIntervalMinutes int,
	pollDelayMs int,
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

	reactionHandler := NewReactionHandler(watchRepo, s)
	threadNotifier := NewThreadNotifier(s, watchRepo)
	pollingWorker := appwatch.NewPollingWorker(watchRepo, threadNotifier, checkIntervalMinutes, pollDelayMs)

	return &Bot{
		state:           s,
		handler:         h,
		reactionHandler: reactionHandler,
		pollingWorker:   pollingWorker,
	}, nil
}

// NewBotWithDeps はテスト用に依存を直接注入する。state が nil のとき Gateway 接続を省略する。
func NewBotWithDeps(s *state.State, h *Handler, rh *ReactionHandler, pw *appwatch.PollingWorker) *Bot {
	return &Bot{
		state:           s,
		handler:         h,
		reactionHandler: rh,
		pollingWorker:   pw,
	}
}

// Run はBotを起動し、Gatewayに接続する。ブロッキング。
func (b *Bot) Run(ctx context.Context) error {
	if b.state != nil {
		b.state.AddHandler(b.handler.HandleMessageCreate)
		b.state.AddHandler(b.reactionHandler.HandleReactionAdd)
		b.state.AddHandler(b.reactionHandler.HandleReactionRemove)

		if err := b.state.Connect(ctx); err != nil {
			return err
		}
	}

	log.Println("[yahoo_auctions_bot] Bot started")

	go b.pollingWorker.Start(ctx)

	<-ctx.Done()
	return ctx.Err()
}
