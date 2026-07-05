package discord

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

// SessionAPI は Embed 送信・スレッド・メッセージ取得・Bot 自身の ID 取得に使う Discord API。
// *state.State が満たす。
type SessionAPI interface {
	SendMessageComplex(channelID discord.ChannelID, data api.SendMessageData) (*discord.Message, error)
	EditMessageComplex(channelID discord.ChannelID, messageID discord.MessageID, data api.EditMessageData) (*discord.Message, error)
	StartThreadWithMessage(channelID discord.ChannelID, messageID discord.MessageID, data api.StartThreadData) (*discord.Channel, error)
	Message(channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error)
	Me() (*discord.User, error)
}
