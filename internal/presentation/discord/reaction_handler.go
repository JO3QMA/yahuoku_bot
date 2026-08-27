package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"

	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	infralisting "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/listing"
)

const (
	watchEmoji = "\U0001F514" // 🔔
	eyesEmoji  = "\U0001F440" // 👀
)

// ReactionHandler はリアクションイベントを処理し、Watch の登録/解除を行う。
type ReactionHandler struct {
	watchUsecase  *appwatch.WatchUsecase
	listingClient infralisting.Client
	api           SessionAPI
}

// NewReactionHandler はReactionHandlerを生成する。
func NewReactionHandler(
	watchUsecase *appwatch.WatchUsecase,
	listingClient infralisting.Client,
	api SessionAPI,
) *ReactionHandler {
	return &ReactionHandler{
		watchUsecase:  watchUsecase,
		listingClient: listingClient,
		api:           api,
	}
}

func (h *ReactionHandler) botUserID() discord.UserID {
	me, err := h.api.Me()
	if err != nil {
		return 0
	}
	return me.ID
}

// HandleReactionAdd はリアクション追加イベントを処理する。
func (h *ReactionHandler) HandleReactionAdd(e *gateway.MessageReactionAddEvent) {
	if !isWatchEmoji(e.Emoji) || e.UserID == h.botUserID() {
		return
	}

	ctx := context.Background()

	msg, err := h.api.Message(e.ChannelID, e.MessageID)
	if err != nil {
		log.Printf("[ReactionHandler] fetch message: %v", err)
		return
	}

	botID := h.botUserID()
	if msg.Author.ID != botID || len(msg.Embeds) == 0 {
		return
	}

	ref, ok := extractListingRefFromEmbeds(msg.Embeds)
	if !ok {
		return
	}

	data, err := h.listingClient.Get(ctx, ref)
	if err != nil {
		log.Printf("[ReactionHandler] GetListing %s/%s: %v", ref.Market, ref.ListingID, err)
		return
	}

	guildID := ""
	if e.GuildID.IsValid() {
		guildID = e.GuildID.String()
	}

	err = h.watchUsecase.Register(
		ctx,
		ref.Market,
		ref.ListingID,
		e.UserID.String(),
		guildID,
		e.ChannelID.String(),
		e.MessageID.String(),
		data.Price,
		data.EndTime,
	)
	if err != nil {
		log.Printf("[ReactionHandler] register watch: %v", err)
		return
	}

	log.Printf("[ReactionHandler] user %s started watching %s/%s (price=%d)", e.UserID, ref.Market, ref.ListingID, data.Price)
}

// HandleReactionRemove はリアクション削除イベントを処理する。
func (h *ReactionHandler) HandleReactionRemove(e *gateway.MessageReactionRemoveEvent) {
	if !isWatchEmoji(e.Emoji) || e.UserID == h.botUserID() {
		return
	}

	ctx := context.Background()

	msg, err := h.api.Message(e.ChannelID, e.MessageID)
	if err != nil {
		log.Printf("[ReactionHandler] fetch message: %v", err)
		return
	}

	botID := h.botUserID()
	if msg.Author.ID != botID || len(msg.Embeds) == 0 {
		return
	}

	ref, ok := extractListingRefFromEmbeds(msg.Embeds)
	if !ok {
		return
	}

	err = h.watchUsecase.Unregister(ctx, ref.Market, ref.ListingID, e.UserID.String(), e.MessageID.String())
	if err != nil {
		log.Printf("[ReactionHandler] unregister watch: %v", err)
		return
	}

	log.Printf("[ReactionHandler] user %s stopped watching %s/%s", e.UserID, ref.Market, ref.ListingID)
}

func isWatchEmoji(emoji discord.Emoji) bool {
	return emoji.Name == watchEmoji || emoji.Name == eyesEmoji
}

func extractListingRefFromEmbeds(embeds []discord.Embed) (listing.Ref, bool) {
	for _, emb := range embeds {
		if ref, ok := listing.ParseRefFromURL(string(emb.URL)); ok {
			return ref, true
		}
	}
	return listing.Ref{}, false
}
