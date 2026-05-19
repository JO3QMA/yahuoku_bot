package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"

	"jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// EmbedSender はメッセージ送信に必要なDiscord APIのインターフェース。*state.State が満たす。
type EmbedSender interface {
	SendMessageComplex(channelID discord.ChannelID, data api.SendMessageData) (*discord.Message, error)
}

// EmbedBuilder はAuctionPreviewからDiscord Embedを構築・送信する。
type EmbedBuilder struct {
	sender EmbedSender
}

// NewEmbedBuilder はEmbedBuilderを生成する。
func NewEmbedBuilder(sender EmbedSender) *EmbedBuilder {
	return &EmbedBuilder{sender: sender}
}

// Build はAuctionPreviewからDiscord Embedを構築する。
func (b *EmbedBuilder) Build(preview *auction.AuctionPreview) discord.Embed {
	emb := discord.NewEmbed()
	emb.Title = preview.Title
	emb.URL = discord.URL(preview.URL)
	emb.Color = 0x7B68EE // ヤフオク風の紫

	if len(preview.Images) > 0 {
		emb.Thumbnail = &discord.EmbedThumbnail{URL: discord.URL(preview.Images[0])}
	}

	fields := []discord.EmbedField{}

	// 現在価格
	priceStr := formatPrice(preview.CurrentPrice)
	fields = append(fields, discord.EmbedField{Name: "現在価格", Value: priceStr, Inline: true})

	// 残り時間
	timeStr := formatEndTime(preview.EndTime)
	fields = append(fields, discord.EmbedField{Name: "残り時間", Value: timeStr, Inline: true})

	// 商品状態
	condition := "不明"
	if preview.Product != nil && preview.Product.Condition != "" {
		condition = preview.Product.Condition
	}
	fields = append(fields, discord.EmbedField{Name: "商品状態", Value: condition, Inline: true})

	// 送料
	shippingStr := formatShipping(preview.Product)
	fields = append(fields, discord.EmbedField{Name: "送料", Value: shippingStr, Inline: true})

	// 商品ジャンル
	if preview.Product != nil {
		fields = append(fields, discord.EmbedField{
			Name: "商品ジャンル", Value: preview.Product.Category.DisplayName(), Inline: true,
		})
	}

	// ジャンル別テンプレート項目
	productFields := formatProductFields(preview.Product)
	fields = append(fields, productFields...)

	emb.Fields = fields
	return *emb
}

// Send は構築済みEmbedを、元メッセージへのリプライとして送信する。
func (b *EmbedBuilder) Send(e *gateway.MessageCreateEvent, emb discord.Embed) (*discord.Message, error) {
	return b.sender.SendMessageComplex(e.ChannelID, api.SendMessageData{
		Embeds: []discord.Embed{emb},
		Reference: &discord.MessageReference{
			MessageID: e.ID,
			ChannelID: e.ChannelID,
			GuildID:   e.GuildID,
		},
		AllowedMentions: &api.AllowedMentions{
			RepliedUser: option.False,
		},
	})
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

func formatShipping(p *product.ProductDetail) string {
	if p == nil || p.ShippingFree == nil {
		return "不明"
	}
	if *p.ShippingFree {
		return "送料無料"
	}
	return "落札者負担"
}

// formatProductFields はジャンル別テンプレートに従い EmbedField を返す。空・不明の項目は含めない。
func formatProductFields(p *product.ProductDetail) []discord.EmbedField {
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
