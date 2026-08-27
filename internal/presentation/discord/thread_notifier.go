package discord

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"

	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

const maxThreadNameLen = 100

// ThreadNotifier は NotificationThread 経由で PriceAlert / EndingReminder を送信する。
type ThreadNotifier struct {
	api  SessionAPI
	repo domainwatch.Repository
}

// NewThreadNotifier はThreadNotifierを生成する。
func NewThreadNotifier(api SessionAPI, repo domainwatch.Repository) *ThreadNotifier {
	return &ThreadNotifier{api: api, repo: repo}
}

// NotifyPriceAlert は PriceAlert を NotificationThread に送信する。
func (n *ThreadNotifier) NotifyPriceAlert(ctx context.Context, item *domainwatch.Watch, oldPrice, newPrice int64, title string) error {
	threadID, err := n.ensureThread(ctx, item, title)
	if err != nil {
		return fmt.Errorf("ensure thread: %w", err)
	}
	return n.sendPriceAlert(threadID, item, oldPrice, newPrice)
}

func (n *ThreadNotifier) sendPriceAlert(threadID discord.ChannelID, item *domainwatch.Watch, oldPrice, newPrice int64) error {
	content := fmt.Sprintf(
		"<@%s> 価格が上昇しました: ¥%s → ¥%s",
		item.UserID,
		formatIntWithComma(oldPrice),
		formatIntWithComma(newPrice),
	)

	_, err := n.api.SendMessageComplex(threadID, api.SendMessageData{
		Content: content,
		AllowedMentions: &api.AllowedMentions{
			Users: []discord.UserID{discord.UserID(mustSnowflake(item.UserID))},
		},
	})
	if err != nil {
		return fmt.Errorf("send price notification: %w", err)
	}

	logPriceAlertSent(item.ListingID, item.UserID)
	return nil
}

// logPriceAlertSent は PriceAlert 送信成功をログする（テストでカバーしやすくするため分離）。
func logPriceAlertSent(auctionID, userID string) {
	log.Printf("[ThreadNotifier] price alert sent for auction %s (user=%s)", auctionID, userID)
}

// NotifyEndingReminder は EndingReminder を NotificationThread に送信する。
func (n *ThreadNotifier) NotifyEndingReminder(ctx context.Context, item *domainwatch.Watch, currentPrice int64, title string, remaining time.Duration) error {
	threadID, err := n.ensureThread(ctx, item, title)
	if err != nil {
		return fmt.Errorf("ensure thread: %w", err)
	}
	return n.sendEndingReminder(threadID, item, currentPrice, remaining)
}

func (n *ThreadNotifier) sendEndingReminder(threadID discord.ChannelID, item *domainwatch.Watch, currentPrice int64, remaining time.Duration) error {
	minutes := remainingMinutesForDisplay(remaining)
	content := fmt.Sprintf(
		"<@%s> オークション終了まで残り約%d分です。現在価格: ¥%s",
		item.UserID,
		minutes,
		formatIntWithComma(currentPrice),
	)

	_, err := n.api.SendMessageComplex(threadID, api.SendMessageData{
		Content: content,
		AllowedMentions: &api.AllowedMentions{
			Users: []discord.UserID{discord.UserID(mustSnowflake(item.UserID))},
		},
	})
	if err != nil {
		return fmt.Errorf("send ending notification: %w", err)
	}

	logEndingReminderSent(item.ListingID, item.UserID)
	return nil
}

// remainingMinutesForDisplay は残り時間を通知文言用の分に切り上げる（1分未満は1分）。
func remainingMinutesForDisplay(d time.Duration) int {
	if d <= 0 {
		return 1
	}
	minutes := int((d + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		return 1
	}
	return minutes
}

func logEndingReminderSent(auctionID, userID string) {
	log.Printf("[ThreadNotifier] ending reminder sent for auction %s (user=%s)", auctionID, userID)
}

func (n *ThreadNotifier) ensureThread(ctx context.Context, item *domainwatch.Watch, title string) (discord.ChannelID, error) {
	// 既存 NotificationThread があればそのまま使う
	if item.ThreadID != "" {
		return discord.ChannelID(mustSnowflake(item.ThreadID)), nil
	}

	// 同一 Preview メッセージの NotificationThread を DB で共有
	siblings, err := n.repo.FindByMessage(ctx, item.MessageID)
	if err == nil {
		for _, s := range siblings {
			if s.ThreadID != "" {
				// 現アイテムにもthread_idを反映
				_ = n.repo.UpdateThreadID(ctx, item.MessageID, s.ThreadID)
				return discord.ChannelID(mustSnowflake(s.ThreadID)), nil
			}
		}
	}

	threadName := truncate("監視通知: "+title, maxThreadNameLen)

	channelID := discord.ChannelID(mustSnowflake(item.ChannelID))
	messageID := discord.MessageID(mustSnowflake(item.MessageID))

	ch, err := n.api.StartThreadWithMessage(channelID, messageID, api.StartThreadData{
		Name:                threadName,
		AutoArchiveDuration: discord.OneDayArchive,
	})
	if err != nil {
		return 0, fmt.Errorf("start thread: %w", err)
	}

	threadID := ch.ID.String()
	if err := n.repo.UpdateThreadID(ctx, item.MessageID, threadID); err != nil {
		log.Printf("[ThreadNotifier] update thread_id in DB: %v", err)
	}

	return ch.ID, nil
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func mustSnowflake(s string) discord.Snowflake {
	sf, _ := discord.ParseSnowflake(s)
	return sf
}
