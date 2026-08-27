package discord

import (
	"testing"

	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"github.com/diamondburned/arikawa/v3/discord"
)

func TestIsWatchEmoji(t *testing.T) {
	tests := []struct {
		name  string
		emoji discord.Emoji
		want  bool
	}{
		{"bell", discord.Emoji{Name: "\U0001F514"}, true},
		{"eyes", discord.Emoji{Name: "\U0001F440"}, true},
		{"other", discord.Emoji{Name: "\U0001F44D"}, false},
		{"empty", discord.Emoji{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWatchEmoji(tt.emoji); got != tt.want {
				t.Errorf("isWatchEmoji() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractListingRefFromEmbeds(t *testing.T) {
	tests := []struct {
		name   string
		embeds []discord.Embed
		want   dlisting.Ref
		ok     bool
	}{
		{
			"valid url",
			[]discord.Embed{{URL: "https://page.auctions.yahoo.co.jp/jp/auction/k1218678393"}},
			dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: "k1218678393"},
			true,
		},
		{
			"no url",
			[]discord.Embed{{URL: "https://example.com"}},
			dlisting.Ref{},
			false,
		},
		{
			"empty embeds",
			nil,
			dlisting.Ref{},
			false,
		},
		{
			"multiple embeds first match",
			[]discord.Embed{
				{URL: "https://example.com"},
				{URL: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678"},
			},
			dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: "abc12345678"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractListingRefFromEmbeds(tt.embeds)
			if ok != tt.ok || got != tt.want {
				t.Errorf("extractListingRefFromEmbeds() = (%#v, %v), want (%#v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
