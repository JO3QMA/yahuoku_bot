package discord

import (
	"context"
	"log"
	"regexp"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"

	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

const watchEmoji = "\U0001F514" // 🔔

var embedURLRe = regexp.MustCompile(`auctions?\.yahoo\.co\.jp/[^/]+/auction/([a-zA-Z0-9]{8,11})`)

// MessageFetcher はメッセージを取得するインターフェース。*state.State が満たす。
type MessageFetcher interface {
	Message(channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error)
}

// BotIdentifier はBot自身のユーザーIDを取得するインターフェース。
type BotIdentifier interface {
	Me() (*discord.User, error)
}

// ReactionHandler はリアクションイベントを処理し、監視の登録/解除を行う。
type ReactionHandler struct {
	watchUsecase  *appwatch.WatchUsecase
	auctionClient infraauction.Client
	fetcher       MessageFetcher
	bot           BotIdentifier
}

// NewReactionHandler はReactionHandlerを生成する。
func NewReactionHandler(
	watchUsecase *appwatch.WatchUsecase,
	auctionClient infraauction.Client,
	fetcher MessageFetcher,
	bot BotIdentifier,
) *ReactionHandler {
	return &ReactionHandler{
		watchUsecase:  watchUsecase,
		auctionClient: auctionClient,
		fetcher:       fetcher,
		bot:           bot,
	}
}

func (h *ReactionHandler) botUserID() discord.UserID {
	me, err := h.bot.Me()
	if err != nil {
		return 0
	}
	return me.ID
}

// HandleReactionAdd はリアクション追加イベントを処理する。
func (h *ReactionHandler) HandleReactionAdd(e *gateway.MessageReactionAddEvent) {
	if !isBellEmoji(e.Emoji) || e.UserID == h.botUserID() {
		return
	}

	ctx := context.Background()

	msg, err := h.fetcher.Message(e.ChannelID, e.MessageID)
	if err != nil {
		log.Printf("[ReactionHandler] fetch message: %v", err)
		return
	}

	botID := h.botUserID()
	if msg.Author.ID != botID || len(msg.Embeds) == 0 {
		return
	}

	auctionID := extractAuctionIDFromEmbeds(msg.Embeds)
	if auctionID == "" {
		return
	}

	data, err := h.auctionClient.GetAuction(ctx, auctionID)
	if err != nil {
		log.Printf("[ReactionHandler] GetAuction %s: %v", auctionID, err)
		return
	}

	guildID := ""
	if e.GuildID.IsValid() {
		guildID = e.GuildID.String()
	}

	err = h.watchUsecase.Register(
		ctx,
		auctionID,
		e.UserID.String(),
		guildID,
		e.ChannelID.String(),
		e.MessageID.String(),
		data.CurrentPrice,
		data.EndTime,
	)
	if err != nil {
		log.Printf("[ReactionHandler] register watch: %v", err)
		return
	}

	log.Printf("[ReactionHandler] user %s started watching auction %s (price=%d)", e.UserID, auctionID, data.CurrentPrice)
}

// HandleReactionRemove はリアクション削除イベントを処理する。
func (h *ReactionHandler) HandleReactionRemove(e *gateway.MessageReactionRemoveEvent) {
	if !isBellEmoji(e.Emoji) || e.UserID == h.botUserID() {
		return
	}

	ctx := context.Background()

	msg, err := h.fetcher.Message(e.ChannelID, e.MessageID)
	if err != nil {
		log.Printf("[ReactionHandler] fetch message: %v", err)
		return
	}

	botID := h.botUserID()
	if msg.Author.ID != botID || len(msg.Embeds) == 0 {
		return
	}

	auctionID := extractAuctionIDFromEmbeds(msg.Embeds)
	if auctionID == "" {
		return
	}

	err = h.watchUsecase.Unregister(ctx, auctionID, e.UserID.String(), e.MessageID.String())
	if err != nil {
		log.Printf("[ReactionHandler] unregister watch: %v", err)
		return
	}

	log.Printf("[ReactionHandler] user %s stopped watching auction %s", e.UserID, auctionID)
}

func isBellEmoji(emoji discord.Emoji) bool {
	return emoji.Name == watchEmoji
}

func extractAuctionIDFromEmbeds(embeds []discord.Embed) string {
	for _, emb := range embeds {
		if m := embedURLRe.FindStringSubmatch(string(emb.URL)); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}
