package discord

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"

	"jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
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
	if preview.Spec != nil && preview.Spec.Condition != "" {
		condition = preview.Spec.Condition
	}
	fields = append(fields, discord.EmbedField{Name: "商品状態", Value: condition, Inline: true})

	// 送料
	shippingStr := formatShipping(preview.Spec)
	fields = append(fields, discord.EmbedField{Name: "送料", Value: shippingStr, Inline: true})

	// PCスペック（各項目を独立して表示）
	specFields := formatSpecFields(preview.Spec)
	fields = append(fields, specFields...)

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

func formatShipping(s *spec.Spec) string {
	if s == nil || s.ShippingFree == nil {
		return "不明"
	}
	if *s.ShippingFree {
		return "送料無料"
	}
	return "落札者負担"
}

// formatSpecFields はSpecの7項目をそれぞれ独立したEmbedFieldとして返す。値が空の項目は含めない。
func formatSpecFields(s *spec.Spec) []discord.EmbedField {
	if s == nil {
		return nil
	}
	fields := []discord.EmbedField{}
	if v := s.CPUModelLine; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "CPU型番 (x個数) (周波数)", Value: v, Inline: false})
	}
	if v := s.CoreThreadInfo; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "CPUコア数/スレッド数", Value: v, Inline: false})
	}
	if s.SocketCount > 0 {
		fields = append(fields, discord.EmbedField{Name: "ソケット数", Value: fmt.Sprintf("%d", s.SocketCount), Inline: true})
	}
	if v := s.MemoryInfo; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "メモリー容量/枚数", Value: v, Inline: true})
	}
	if v := s.StorageType; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "ストレージ種別", Value: v, Inline: true})
	}
	if v := s.StorageCapacity; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "ストレージ容量", Value: v, Inline: true})
	}
	if v := s.OtherNotes; v != "" && v != "不明" {
		fields = append(fields, discord.EmbedField{Name: "その他特記事項", Value: v, Inline: false})
	}
	return fields
}
