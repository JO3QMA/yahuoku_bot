package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jo3qma/sansai"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

const (
	watchEmoji = "\U0001F514" // 🔔
	eyesEmoji  = "\U0001F440" // 👀
)

// sansaiGetItem は sansai.Get の差し替え用（テストのみ、同一 package）。
var sansaiGetItem = sansai.Get

// ReactionHandler はリアクションイベントを処理し、Watch の登録/解除を行う。
type ReactionHandler struct {
	watchRepo domainwatch.Repository
	api       SessionAPI
}

// NewReactionHandler はReactionHandlerを生成する。
func NewReactionHandler(
	watchRepo domainwatch.Repository,
	api SessionAPI,
) *ReactionHandler {
	return &ReactionHandler{
		watchRepo: watchRepo,
		api:       api,
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

	item, err := sansaiGetItem(ctx, sansai.Market(ref.Market), ref.ListingID)
	if err != nil {
		log.Printf("[ReactionHandler] GetListing %s/%s: %v", ref.Market, ref.ListingID, err)
		return
	}
	if item == nil {
		log.Printf("[ReactionHandler] GetListing %s/%s: listing not found", ref.Market, ref.ListingID)
		return
	}
	data := listing.FromSansaiItem(item)

	guildID := ""
	if e.GuildID.IsValid() {
		guildID = e.GuildID.String()
	}

	err = h.watchRepo.Add(ctx, &domainwatch.Watch{
		Market:         ref.Market,
		ListingID:      ref.ListingID,
		UserID:         e.UserID.String(),
		GuildID:        guildID,
		ChannelID:      e.ChannelID.String(),
		MessageID:      e.MessageID.String(),
		LastKnownPrice: data.Price,
		EndTime:        data.EndTime,
	})
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

	err = h.watchRepo.Remove(ctx, ref.Market, ref.ListingID, e.UserID.String(), e.MessageID.String())
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
