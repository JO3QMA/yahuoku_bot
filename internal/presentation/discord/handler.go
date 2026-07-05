package discord

import (
	"context"
	"log"
	"regexp"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"

	"jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
)

// yahooAuctionURLRe はヤフオクURLからオークションIDを抽出する正規表現。
// 例: https://page.auctions.yahoo.co.jp/jp/auction/xxxx1234
var yahooAuctionURLRe = regexp.MustCompile(`auctions?\.yahoo\.co\.jp/[^/]+/auction/([a-zA-Z0-9]{8,11})`)

const (
	handlerPreviewTimeout = 60 * time.Second
	handlerMarketTimeout  = 25 * time.Second
)

// Handler はメッセージを監視し、ヤフオク URL から Preview を生成するハンドラー。
type Handler struct {
	usecase *auction.PreviewUsecase
	market  *appmarket.EstimateUsecase
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
func NewHandler(usecase *auction.PreviewUsecase, market *appmarket.EstimateUsecase, embed *EmbedBuilder, allowed *AllowedFilter) *Handler {
	return &Handler{usecase: usecase, market: market, embed: embed, allowed: allowed}
}

// HandleMessageCreate はMessageCreateEventを処理する。arikawaのAddHandlerに渡す。
func (h *Handler) HandleMessageCreate(e *gateway.MessageCreateEvent) {
	// Bot自身のメッセージは無視
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

	ids := yahooAuctionURLRe.FindAllStringSubmatch(e.Content, -1)
	if len(ids) == 0 {
		return
	}

	seen := make(map[string]bool)
	for _, m := range ids {
		auctionID := m[1]
		if seen[auctionID] {
			continue
		}
		seen[auctionID] = true

		ctx, cancel := context.WithTimeout(context.Background(), handlerPreviewTimeout)
		preview, err := h.usecase.Execute(ctx, auctionID)
		cancel()
		if err != nil {
			log.Printf("[yahoo_auctions_bot] GetAuction %s: %v", auctionID, err)
			continue
		}

		emb := h.embed.Build(preview)
		msg, err := h.embed.Send(e, emb)
		if err != nil {
			log.Printf("[yahoo_auctions_bot] SendMessage: %v", err)
			continue
		}

		if h.market != nil && msg != nil {
			go h.attachMarketEstimate(preview, msg.ID, e.ChannelID)
		}
	}
}

func (h *Handler) attachMarketEstimate(preview *auction.Preview, messageID discord.MessageID, channelID discord.ChannelID) {
	ctx, cancel := context.WithTimeout(context.Background(), handlerMarketTimeout)
	defer cancel()
	est, err := h.market.Execute(ctx, preview.Title, preview.Description, preview.Product)
	if err != nil {
		log.Printf("[yahoo_auctions_bot] MarketEstimate %s: %v", preview.AuctionID, err)
		return
	}
	if est == nil {
		return
	}
	preview.MarketEstimate = est
	emb := h.embed.Build(preview)
	if err := h.embed.Edit(channelID, messageID, emb); err != nil {
		log.Printf("[yahoo_auctions_bot] EditMessage market %s: %v", preview.AuctionID, err)
	}
}
