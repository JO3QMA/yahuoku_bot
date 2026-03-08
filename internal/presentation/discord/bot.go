package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"

	"jo3qma.com/yahoo_auctions_bot/internal/application/auction"
)

// Bot はarikawa Stateとハンドラーを保持し、Discord Botを起動する。
type Bot struct {
	state   *state.State
	handler *Handler
}

// NewBot はBotを生成する。DIは呼び出し元で行う。
func NewBot(token string, usecase *auction.PreviewUsecase, allowed *AllowedFilter) (*Bot, error) {
	s := state.NewWithIntents(token,
		gateway.IntentGuilds,
		gateway.IntentGuildMessages,
		gateway.IntentDirectMessages,
		gateway.IntentMessageContent,
	)

	embed := NewEmbedBuilder(s)
	h := NewHandler(usecase, embed, allowed)
	return &Bot{state: s, handler: h}, nil
}

// Run はBotを起動し、Gatewayに接続する。ブロッキング。
func (b *Bot) Run(ctx context.Context) error {
	b.state.AddHandler(b.handler.HandleMessageCreate)

	if err := b.state.Connect(ctx); err != nil {
		return err
	}

	log.Println("[yahoo_auctions_bot] Bot started")
	<-ctx.Done()
	return ctx.Err()
}
