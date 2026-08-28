package discord

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
)

// Handler はメッセージを監視し、C2C マーケット URL から Preview を生成するハンドラー。
type Handler struct {
	usecase *applisting.PreviewUsecase
	embed   *EmbedBuilder
	allowed *AllowedFilter
}

// AllowedFilter はconfigに基づくサーバー・チャンネルフィルタ。
type AllowedFilter struct {
	Guilds   []string
	Channels []string
}

// NewAllowedFilter はAllowedFilterを生成する。
func NewAllowedFilter(guilds, channels []string) *AllowedFilter {
	return &AllowedFilter{Guilds: guilds, Channels: channels}
}

// Allow は指定のguildID/channelIDが許可されているか返す。
func (f *AllowedFilter) Allow(guildID, channelID string) bool {
	if len(f.Guilds) > 0 {
		ok := false
		for _, g := range f.Guilds {
			if g == guildID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Channels) > 0 {
		ok := false
		for _, c := range f.Channels {
			if c == channelID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// NewHandler はHandlerを生成する。
func NewHandler(usecase *applisting.PreviewUsecase, embed *EmbedBuilder, allowed *AllowedFilter) *Handler {
	return &Handler{usecase: usecase, embed: embed, allowed: allowed}
}

// HandleMessageCreate はMessageCreateEventを処理する。arikawaのAddHandlerに渡す。
func (h *Handler) HandleMessageCreate(e *gateway.MessageCreateEvent) {
	if e.Author.Bot {
		return
	}

	guildID := ""
	if e.GuildID.IsValid() {
		guildID = e.GuildID.String()
	}
	channelID := e.ChannelID.String()

	if !h.allowed.Allow(guildID, channelID) {
		return
	}

	refs := listing.ParseRefs(e.Content)
	if len(refs) == 0 {
		return
	}

	seen := make(map[listing.Ref]bool)
	for _, ref := range refs {
		if seen[ref] {
			continue
		}
		seen[ref] = true

		ctx := context.Background()
		preview, err := h.usecase.Execute(ctx, ref)
		if err != nil {
			log.Printf("[yahoo_auctions_bot] GetListing %s/%s: %v", ref.Market, ref.ListingID, err)
			continue
		}

		emb := h.embed.Build(preview)
		_, err = h.embed.Send(e, emb)
		if err != nil {
			log.Printf("[yahoo_auctions_bot] SendMessage: %v", err)
		}
	}
}
