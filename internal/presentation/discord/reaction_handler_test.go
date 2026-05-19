package discord

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
)

func TestIsBellEmoji(t *testing.T) {
	tests := []struct {
		name  string
		emoji discord.Emoji
		want  bool
	}{
		{"bell", discord.Emoji{Name: "\U0001F514"}, true},
		{"other", discord.Emoji{Name: "\U0001F44D"}, false},
		{"empty", discord.Emoji{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBellEmoji(tt.emoji); got != tt.want {
				t.Errorf("isBellEmoji() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractAuctionIDFromEmbeds(t *testing.T) {
	tests := []struct {
		name   string
		embeds []discord.Embed
		want   string
	}{
		{
			"valid url",
			[]discord.Embed{{URL: "https://page.auctions.yahoo.co.jp/jp/auction/k1218678393"}},
			"k1218678393",
		},
		{
			"no url",
			[]discord.Embed{{URL: "https://example.com"}},
			"",
		},
		{
			"empty embeds",
			nil,
			"",
		},
		{
			"multiple embeds first match",
			[]discord.Embed{
				{URL: "https://example.com"},
				{URL: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678"},
			},
			"abc12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAuctionIDFromEmbeds(tt.embeds)
			if got != tt.want {
				t.Errorf("extractAuctionIDFromEmbeds() = %q, want %q", got, tt.want)
			}
		})
	}
}
