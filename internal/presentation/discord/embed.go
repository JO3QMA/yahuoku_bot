package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// EmbedBuilder はPreviewからDiscord Embedを構築・送信する。
type EmbedBuilder struct {
	api SessionAPI
}

// NewEmbedBuilder はEmbedBuilderを生成する。
func NewEmbedBuilder(api SessionAPI) *EmbedBuilder {
	return &EmbedBuilder{api: api}
}

// Build はPreviewからDiscord Embedを構築する。
func (b *EmbedBuilder) Build(preview *applisting.Preview) discord.Embed {
	emb := discord.NewEmbed()
	emb.Title = preview.Title
	emb.URL = discord.URL(preview.URL)
	emb.Color = 0x7B68EE

	if len(preview.Images) > 0 {
		emb.Thumbnail = &discord.EmbedThumbnail{URL: discord.URL(preview.Images[0])}
	}

	fields := []discord.EmbedField{
		{Name: "マーケット", Value: marketDisplayName(preview.Ref.Market), Inline: true},
	}

	if showSaleType(preview) {
		fields = append(fields, discord.EmbedField{Name: "販売形式", Value: "オークション", Inline: true})
	}

	fields = append(fields,
		discord.EmbedField{Name: "現在価格", Value: formatPrice(preview.CurrentPrice), Inline: true},
	)

	if preview.SaleType == listing.SaleTypeAuction {
		fields = append(fields, discord.EmbedField{Name: "残り時間", Value: formatEndTime(preview.EndTime), Inline: true})
	}

	condition := "不明"
	if preview.Product != nil && preview.Product.Condition != "" {
		condition = preview.Product.Condition
	}
	fields = append(fields, discord.EmbedField{Name: "商品状態", Value: condition, Inline: true})

	shippingStr := formatShipping(preview.Product)
	fields = append(fields, discord.EmbedField{Name: "送料", Value: shippingStr, Inline: true})

	if preview.Product != nil {
		fields = append(fields, discord.EmbedField{
			Name: "商品ジャンル", Value: preview.Product.Category.DisplayName(), Inline: true,
		})
	}

	fields = append(fields, formatProductFields(preview.Product)...)

	emb.Fields = fields
	return *emb
}

// Send は構築済みEmbedをチャンネルに通常投稿する（リプライ・スレッドは使わない）。
func (b *EmbedBuilder) Send(e *gateway.MessageCreateEvent, emb discord.Embed) (*discord.Message, error) {
	return b.api.SendMessageComplex(e.ChannelID, api.SendMessageData{
		Embeds: []discord.Embed{emb},
	})
}

func marketDisplayName(m listing.Market) string {
	switch m {
	case listing.MarketYahooAuction:
		return "ヤフオク"
	case listing.MarketYahooFlea:
		return "Yahoo!フリマ"
	case listing.MarketMercari:
		return "メルカリ"
	default:
		return string(m)
	}
}

func showSaleType(preview *applisting.Preview) bool {
	return preview.SaleType == listing.SaleTypeAuction && preview.Ref.Market != listing.MarketYahooAuction
}

func formatPrice(price int64) string {
	if price <= 0 {
		return "不明"
	}
	return fmt.Sprintf("¥%s", formatIntWithComma(price))
}

func formatIntWithComma(n int64) string {
	if n < 0 {
		return "-" + formatIntWithComma(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatIntWithComma(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}

func formatEndTime(endTime *time.Time) string {
	if endTime == nil {
		return "不明"
	}
	now := time.Now()
	if endTime.Before(now) {
		return "終了"
	}
	d := endTime.Sub(now)
	if d < time.Hour {
		return fmt.Sprintf("%d分", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d時間%d分", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%d日", int(d.Hours()/24))
}

func formatShipping(p *product.Product) string {
	if p == nil || p.FreeShipping == nil {
		return "不明"
	}
	if *p.FreeShipping {
		return "送料無料"
	}
	return "落札者負担"
}

func formatProductFields(p *product.Product) []discord.EmbedField {
	if p == nil {
		return nil
	}
	defs := product.TemplatesFor(p.Category)
	values := product.FieldValueMap(p.Fields)
	fields := []discord.EmbedField{}

	for _, def := range defs {
		v, ok := values[def.Key]
		if !ok || v == "" || v == "不明" {
			continue
		}
		display := formatFieldDisplayValue(def.Key, v)
		if display == "" {
			continue
		}
		fields = append(fields, discord.EmbedField{
			Name: def.Label, Value: display, Inline: def.Inline,
		})
	}
	return fields
}

func formatFieldDisplayValue(key, value string) string {
	if key == "socket_count" {
		value = strings.TrimSpace(value)
		if value == "" || value == "0" {
			return ""
		}
	}
	return value
}
