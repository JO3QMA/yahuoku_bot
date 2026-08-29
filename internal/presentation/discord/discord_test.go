package discord

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type mockGW struct {
	connectErr error
}

func (m *mockGW) AddHandler(interface{}) func() { return func() {} }

func (m *mockGW) Connect(ctx context.Context) error { return m.connectErr }

type stubSessionAPI struct {
	sendMsg  *discord.Message
	sendErr  error
	lastSend api.SendMessageData
	msg      *discord.Message
	msgErr   error
	user     *discord.User
	meErr    error
	thread   *discord.Channel
	startErr error
}

func (s *stubSessionAPI) SendMessageComplex(channelID discord.ChannelID, data api.SendMessageData) (*discord.Message, error) {
	s.lastSend = data
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	if s.sendMsg != nil {
		return s.sendMsg, nil
	}
	return &discord.Message{}, nil
}

func (s *stubSessionAPI) Message(channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {
	return s.msg, s.msgErr
}

func (s *stubSessionAPI) Me() (*discord.User, error) {
	return s.user, s.meErr
}

func (s *stubSessionAPI) StartThreadWithMessage(channelID discord.ChannelID, messageID discord.MessageID, data api.StartThreadData) (*discord.Channel, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	if s.thread != nil {
		return s.thread, nil
	}
	return &discord.Channel{ID: 99}, nil
}

type stubListing struct {
	data *dlisting.Data
	err  error
}

func (s *stubListing) Get(ctx context.Context, ref dlisting.Ref) (*dlisting.Data, error) {
	return s.data, s.err
}

type memWatchRepo struct {
	addErr error
	remErr error
	items  []*domainwatch.Watch
}

func (m *memWatchRepo) Add(ctx context.Context, item *domainwatch.Watch) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.items = append(m.items, item)
	return nil
}
func (m *memWatchRepo) Remove(ctx context.Context, market dlisting.Market, listingID, userID, messageID string) error {
	return m.remErr
}
func (m *memWatchRepo) ListActive(ctx context.Context) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (m *memWatchRepo) UpdatePrice(ctx context.Context, id int64, newPrice int64) error { return nil }
func (m *memWatchRepo) MarkReminded(ctx context.Context, id int64) error                 { return nil }
func (m *memWatchRepo) UpdateThreadID(ctx context.Context, messageID, threadID string) error {
	return nil
}
func (m *memWatchRepo) FindByMessage(ctx context.Context, messageID string) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (m *memWatchRepo) RemoveByListing(ctx context.Context, market dlisting.Market, listingID string) error { return nil }

type stubPreviewFetch struct {
	data *dlisting.Data
	err  error
}

func (s *stubPreviewFetch) Get(ctx context.Context, ref dlisting.Ref) (*dlisting.Data, error) {
	return s.data, s.err
}

type stubProductExt struct {
	pd  *product.Product
	err error
}

func (s *stubProductExt) Extract(ctx context.Context, title, description string, imageURLs []string) (*product.Product, error) {
	return s.pd, s.err
}

func testListingData(id string, end *time.Time) *dlisting.Data {
	return &dlisting.Data{
		Ref:         dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: id},
		Title:       "T",
		Price:       1,
		Description: "d",
		EndTime:     end,
		SaleType:    dlisting.SaleTypeAuction,
		IsActive:    true,
	}
}

func TestBot_Run_connectError(t *testing.T) {
	end := time.Now()
	pu := applisting.NewPreviewUsecase(&stubPreviewFetch{data: testListingData("a", &end)}, &stubProductExt{pd: &product.Product{}})
	h := NewHandler(pu, NewEmbedBuilder(&stubSessionAPI{}), NewAllowedFilter(nil, nil))
	repo := &memWatchRepo{}
	wu := appwatch.NewWatchUsecase(repo)
	rh := NewReactionHandler(wu, &stubListing{data: testListingData("a", nil)}, &stubSessionAPI{user: &discord.User{ID: discord.UserID(7)}})
	pw := appwatch.NewPollingWorker(repo, &stubListing{}, &noopNotifier{}, 60, 1)
	b := NewBotWithDeps(&mockGW{connectErr: errors.New("nope")}, h, rh, pw)
	err := b.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

type noopNotifier struct{}

func (n *noopNotifier) NotifyPriceAlert(context.Context, *domainwatch.Watch, int64, int64, string) error {
	return nil
}
func (n *noopNotifier) NotifyEndingReminder(context.Context, *domainwatch.Watch, int64, string, time.Duration) error {
	return nil
}

func TestBot_Run_cancel(t *testing.T) {
	end := time.Now()
	pu := applisting.NewPreviewUsecase(&stubPreviewFetch{data: testListingData("a", &end)}, &stubProductExt{pd: &product.Product{}})
	h := NewHandler(pu, NewEmbedBuilder(&stubSessionAPI{}), NewAllowedFilter(nil, nil))
	repo := &memWatchRepo{}
	wu := appwatch.NewWatchUsecase(repo)
	rh := NewReactionHandler(wu, &stubListing{data: testListingData("a", nil)}, &stubSessionAPI{user: &discord.User{ID: discord.UserID(7)}})
	pw := appwatch.NewPollingWorker(repo, &stubListing{}, &noopNotifier{}, 60, 10_000)
	b := NewBotWithDeps(&mockGW{}, h, rh, pw)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = b.Run(ctx)
}

func TestAllowedFilter_Allow(t *testing.T) {
	f := NewAllowedFilter([]string{"g1"}, []string{"c1"})
	if !f.Allow("g1", "c1") {
		t.Fatal("allow")
	}
	if f.Allow("g2", "c1") {
		t.Fatal("guild")
	}
	if f.Allow("g1", "c2") {
		t.Fatal("channel")
	}
	f2 := NewAllowedFilter(nil, []string{"c1"})
	if !f2.Allow("any", "c1") || f2.Allow("any", "x") {
		t.Fatal("guild wildcard")
	}
	f3 := NewAllowedFilter([]string{"g1"}, nil)
	if !f3.Allow("g1", "any") || f3.Allow("g2", "any") {
		t.Fatal("channel wildcard")
	}
}

func TestHandler_HandleMessageCreate(t *testing.T) {
	end := time.Now()
	data := testListingData("abc12345678", &end)
	pu := applisting.NewPreviewUsecase(&stubPreviewFetch{data: data}, &stubProductExt{pd: &product.Product{}})
	sender := &stubSessionAPI{sendMsg: &discord.Message{ID: 1}}
	h := NewHandler(pu, NewEmbedBuilder(sender), NewAllowedFilter(nil, nil))

	t.Run("ignore bot", func(t *testing.T) {
		h.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{Author: discord.User{Bot: true}, Content: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678"},
		})
	})
	t.Run("not allowed", func(t *testing.T) {
		f := NewAllowedFilter([]string{"g"}, []string{"c"})
		h2 := NewHandler(pu, NewEmbedBuilder(sender), f)
		h2.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Author:    discord.User{Bot: false},
				Content:   "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
				GuildID:   discord.GuildID(9),
				ChannelID: discord.ChannelID(2),
			},
		})
	})
	t.Run("no url", func(t *testing.T) {
		h.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{Author: discord.User{}, Content: "hello"},
		})
	})
	t.Run("usecase error", func(t *testing.T) {
		pu2 := applisting.NewPreviewUsecase(&stubPreviewFetch{err: errors.New("e")}, &stubProductExt{})
		h2 := NewHandler(pu2, NewEmbedBuilder(sender), NewAllowedFilter(nil, nil))
		h2.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Author: discord.User{}, Content: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
			},
		})
	})
	t.Run("send error", func(t *testing.T) {
		pu2 := applisting.NewPreviewUsecase(&stubPreviewFetch{data: data}, &stubProductExt{pd: &product.Product{}})
		h2 := NewHandler(pu2, NewEmbedBuilder(&stubSessionAPI{sendErr: errors.New("s")}), NewAllowedFilter(nil, nil))
		h2.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Author: discord.User{}, Content: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
			},
		})
	})
	t.Run("success", func(t *testing.T) {
		h.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Author: discord.User{}, Content: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
			},
		})
	})
	t.Run("dedupe ids", func(t *testing.T) {
		h.HandleMessageCreate(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Author: discord.User{},
				Content: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678 https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
			},
		})
	})
}

func TestEmbedBuilder_Build_and_Send(t *testing.T) {
	sf := true
	p := &applisting.Preview{
		Ref: dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: "a"}, Title: "T", URL: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678",
		CurrentPrice: 0, Images: []string{"https://i"}, EndTime: nil,
		Product: &product.Product{
			Category: product.CategoryServer, Condition: "新品", FreeShipping: &sf,
			Fields: []product.Field{
				{Key: "server_model", Value: "Fujitsu Primergy RX1330 M4"},
				{Key: "cpu_model_line", Value: "Intel Core Ultra 7 355 @4.25GHz x1"},
				{Key: "core_thread_info", Value: "x"},
				{Key: "socket_count", Value: "2"},
				{Key: "memory_info", Value: "DDR4 Unbuffered 2133MHz 8GB x8 Total: 64GB"},
				{Key: "storage_info", Value: "SSD 256GB x1"},
				{Key: "gpu", Value: "AMD Radeon RX9070XT x2"},
				{Key: "other_notes", Value: "o"},
			},
		},
	}
	sender := &stubSessionAPI{}
	b := NewEmbedBuilder(sender)
	emb := b.Build(p)
	if len(emb.Fields) == 0 {
		t.Fatal("fields")
	}
	e := &gateway.MessageCreateEvent{}
	e.ChannelID = 1
	e.ID = 2
	_, err := b.Send(e, emb)
	if err != nil {
		t.Fatal(err)
	}
	if sender.lastSend.Reference != nil {
		t.Fatal("embed must not be sent as a reply")
	}
}

func TestEmbedBuilder_priceAndTime(t *testing.T) {
	b := NewEmbedBuilder(&stubSessionAPI{})
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(30 * time.Minute)
	future2 := time.Now().Add(3 * time.Hour)
	future3 := time.Now().Add(30 * time.Hour)
	_ = b.Build(&applisting.Preview{CurrentPrice: -1, EndTime: &past})
	_ = b.Build(&applisting.Preview{CurrentPrice: 1500, EndTime: &future})
	_ = b.Build(&applisting.Preview{CurrentPrice: 2000, EndTime: &future2})
	_ = b.Build(&applisting.Preview{CurrentPrice: 3000, EndTime: &future3})
	sf := false
	_ = b.Build(&applisting.Preview{Product: &product.Product{FreeShipping: &sf}})
	_ = b.Build(&applisting.Preview{Product: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "不明"}},
	}})
	_ = b.Build(&applisting.Preview{CurrentPrice: 12_345_678, EndTime: &future})
}

func TestThreadNotifier(t *testing.T) {
	ctx := context.Background()
	bid := discord.Snowflake(100)
	uid := discord.Snowflake(200)
	repo := &memWatchRepo{}
	api := &stubSessionAPI{}
	n := NewThreadNotifier(api, repo)
	item := &domainwatch.Watch{
		UserID:    uid.String(),
		ChannelID: bid.String(),
		MessageID: bid.String(),
		ListingID: "a",
		ThreadID:  "99",
	}
	if err := n.NotifyPriceAlert(ctx, item, 10, 20, "title"); err != nil {
		t.Fatal(err)
	}
	item2 := &domainwatch.Watch{
		UserID: uid.String(), ChannelID: bid.String(), MessageID: bid.String(),
		ListingID: "a2", ThreadID: "",
	}
	if err := n.NotifyEndingReminder(ctx, item2, 5, "t2", 5*time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestThreadNotifier_ensureThread_sibling(t *testing.T) {
	ctx := context.Background()
	repo := &siblingRepo{}
	api := &stubSessionAPI{}
	n := NewThreadNotifier(api, repo)
	item := &domainwatch.Watch{
		UserID: "1", ChannelID: "10", MessageID: "20", ListingID: "a",
		ThreadID: "",
	}
	if err := n.NotifyPriceAlert(ctx, item, 1, 2, "long title for truncate test long long long"); err != nil {
		t.Fatal(err)
	}
}

type siblingRepo struct{}

func (s *siblingRepo) Add(ctx context.Context, item *domainwatch.Watch) error { return nil }
func (s *siblingRepo) Remove(ctx context.Context, market dlisting.Market, listingID, userID, messageID string) error {
	return nil
}
func (s *siblingRepo) ListActive(ctx context.Context) ([]*domainwatch.Watch, error) { return nil, nil }
func (s *siblingRepo) UpdatePrice(ctx context.Context, id int64, newPrice int64) error   { return nil }
func (s *siblingRepo) MarkReminded(ctx context.Context, id int64) error                { return nil }
func (s *siblingRepo) UpdateThreadID(ctx context.Context, messageID, threadID string) error {
	return nil
}
func (s *siblingRepo) FindByMessage(ctx context.Context, messageID string) ([]*domainwatch.Watch, error) {
	return []*domainwatch.Watch{{ThreadID: "55"}}, nil
}
func (s *siblingRepo) RemoveByListing(ctx context.Context, market dlisting.Market, listingID string) error { return nil }

func TestThreadNotifier_startThreadErr(t *testing.T) {
	ctx := context.Background()
	repo := &memWatchRepo{}
	api := &stubSessionAPI{startErr: errors.New("st")}
	n := NewThreadNotifier(api, repo)
	item := &domainwatch.Watch{
		UserID: "1", ChannelID: "10", MessageID: "20", ListingID: "a",
	}
	err := n.NotifyPriceAlert(ctx, item, 1, 2, "t")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestThreadNotifier_startThreadErr_endingSoon(t *testing.T) {
	ctx := context.Background()
	n := NewThreadNotifier(&stubSessionAPI{startErr: errors.New("st")}, &memWatchRepo{})
	item := &domainwatch.Watch{
		UserID: "200", ChannelID: "100", MessageID: "20", ListingID: "a", ThreadID: "",
	}
	err := n.NotifyEndingReminder(ctx, item, 1, "t", time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestThreadNotifier_sendErr(t *testing.T) {
	ctx := context.Background()
	repo := &memWatchRepo{}
	api := &stubSessionAPI{sendErr: errors.New("se")}
	n := NewThreadNotifier(api, repo)
	item := &domainwatch.Watch{UserID: "200", ChannelID: "100", MessageID: "20", ThreadID: "99"}
	err := n.NotifyEndingReminder(ctx, item, 1, "t", time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

type findErrThreadRepo struct{}

func (findErrThreadRepo) Add(context.Context, *domainwatch.Watch) error { return nil }
func (findErrThreadRepo) Remove(context.Context, dlisting.Market, string, string, string) error { return nil }
func (findErrThreadRepo) ListActive(context.Context) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (findErrThreadRepo) UpdatePrice(context.Context, int64, int64) error { return nil }
func (findErrThreadRepo) MarkReminded(context.Context, int64) error { return nil }
func (findErrThreadRepo) UpdateThreadID(context.Context, string, string) error { return nil }
func (findErrThreadRepo) FindByMessage(context.Context, string) ([]*domainwatch.Watch, error) {
	return nil, errors.New("find")
}
func (findErrThreadRepo) RemoveByListing(context.Context, dlisting.Market, string) error { return nil }

func TestThreadNotifier_findByMessageErrStillCreates(t *testing.T) {
	ctx := context.Background()
	n := NewThreadNotifier(&stubSessionAPI{}, findErrThreadRepo{})
	item := &domainwatch.Watch{
		UserID: "200", ChannelID: "100", MessageID: "20", ListingID: "a", ThreadID: "",
	}
	if err := n.NotifyPriceAlert(ctx, item, 1, 2, "t"); err != nil {
		t.Fatal(err)
	}
}

type updateThreadErrRepo struct{}

func (updateThreadErrRepo) Add(context.Context, *domainwatch.Watch) error { return nil }
func (updateThreadErrRepo) Remove(context.Context, dlisting.Market, string, string, string) error { return nil }
func (updateThreadErrRepo) ListActive(context.Context) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (updateThreadErrRepo) UpdatePrice(context.Context, int64, int64) error { return nil }
func (updateThreadErrRepo) MarkReminded(context.Context, int64) error { return nil }
func (updateThreadErrRepo) FindByMessage(context.Context, string) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (updateThreadErrRepo) UpdateThreadID(context.Context, string, string) error {
	return errors.New("db")
}
func (updateThreadErrRepo) RemoveByListing(context.Context, dlisting.Market, string) error { return nil }

func TestThreadNotifier_updateThreadIDLogNonFatal(t *testing.T) {
	ctx := context.Background()
	n := NewThreadNotifier(&stubSessionAPI{}, updateThreadErrRepo{})
	item := &domainwatch.Watch{
		UserID: "200", ChannelID: "100", MessageID: "20", ListingID: "a", ThreadID: "",
	}
	if err := n.NotifyPriceAlert(ctx, item, 1, 2, "t"); err != nil {
		t.Fatal(err)
	}
}

func TestThreadNotifier_negativePriceComma(t *testing.T) {
	ctx := context.Background()
	n := NewThreadNotifier(&stubSessionAPI{}, &memWatchRepo{})
	item := &domainwatch.Watch{
		UserID: "200", ChannelID: "100", MessageID: "20", ThreadID: "99",
	}
	if err := n.NotifyPriceAlert(ctx, item, -1200, -500, "t"); err != nil {
		t.Fatal(err)
	}
}

func TestThreadNotifier_sendPriceAlert_direct(t *testing.T) {
	n := NewThreadNotifier(&stubSessionAPI{}, &memWatchRepo{})
	item := &domainwatch.Watch{UserID: "200", Market: dlisting.MarketYahooAuction, ListingID: "a"}
	if err := n.sendPriceAlert(discord.ChannelID(99), item, 1, 2); err != nil {
		t.Fatal(err)
	}
}

func TestThreadNotifier_sendEndingReminder_direct(t *testing.T) {
	api := &stubSessionAPI{}
	n := NewThreadNotifier(api, &memWatchRepo{})
	item := &domainwatch.Watch{UserID: "200", Market: dlisting.MarketYahooAuction, ListingID: "a"}
	if err := n.sendEndingReminder(discord.ChannelID(99), item, 500, 8*time.Minute+30*time.Second); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(api.lastSend.Content), "残り約9分") {
		t.Fatalf("content %q", api.lastSend.Content)
	}
}

func TestThreadNotifier_sendPriceAlert_sendErr(t *testing.T) {
	n := NewThreadNotifier(&stubSessionAPI{sendErr: errors.New("se")}, &memWatchRepo{})
	item := &domainwatch.Watch{UserID: "200", Market: dlisting.MarketYahooAuction, ListingID: "a"}
	err := n.sendPriceAlert(discord.ChannelID(99), item, 1, 2)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestThreadNotifier_sendEndingReminder_sendErr(t *testing.T) {
	n := NewThreadNotifier(&stubSessionAPI{sendErr: errors.New("se")}, &memWatchRepo{})
	item := &domainwatch.Watch{UserID: "200", Market: dlisting.MarketYahooAuction, ListingID: "a"}
	err := n.sendEndingReminder(discord.ChannelID(99), item, 1, time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestThreadNotifier_successLogLines(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	ctx := context.Background()
	n := NewThreadNotifier(&stubSessionAPI{}, &memWatchRepo{})
	item := &domainwatch.Watch{
		UserID: "200", ChannelID: "100", MessageID: "20", ListingID: "a", ThreadID: "99",
	}
	if err := n.NotifyPriceAlert(ctx, item, 1, 2, "t"); err != nil {
		t.Fatal(err)
	}
	if err := n.NotifyEndingReminder(ctx, item, 500, "t", 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "price alert") || !strings.Contains(out, "ending reminder") {
		t.Fatalf("log output: %q", out)
	}
}

func TestLogThreadNotifyHelpers_smoke(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	logPriceAlertSent("aid", "uid")
	logEndingReminderSent("aid", "uid")
	s := buf.String()
	if !strings.Contains(s, "aid") || !strings.Contains(s, "uid") {
		t.Fatalf("%q", s)
	}
}

func TestReactionHandler_flows(t *testing.T) {
	repo := &memWatchRepo{}
	wu := appwatch.NewWatchUsecase(repo)
	ac := &stubListing{data: testListingData("a", nil)}
	botID := discord.UserID(50)
	ch := discord.ChannelID(10)
	mid := discord.MessageID(20)
	msg := &discord.Message{
		Author: discord.User{ID: botID},
		Embeds: []discord.Embed{{URL: "https://page.auctions.yahoo.co.jp/jp/auction/abc12345678"}},
	}
	sess := &stubSessionAPI{msg: msg, user: &discord.User{ID: botID}}
	rh := NewReactionHandler(wu, ac, sess)

	t.Run("wrong emoji", func(t *testing.T) {
		rh.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 1, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "x"},
		})
	})
	t.Run("me err", func(t *testing.T) {
		rh2 := NewReactionHandler(wu, ac, &stubSessionAPI{msg: msg, meErr: errors.New("m")})
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 1, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("bot self", func(t *testing.T) {
		rh.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: botID, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("fetch err", func(t *testing.T) {
		rh2 := NewReactionHandler(wu, ac, &stubSessionAPI{msgErr: errors.New("f"), user: &discord.User{ID: botID}})
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("non bot author", func(t *testing.T) {
		rh2 := NewReactionHandler(wu, ac, &stubSessionAPI{msg: &discord.Message{
			Author: discord.User{ID: 2}, Embeds: msg.Embeds,
		}, user: &discord.User{ID: botID}})
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("no auction in embed", func(t *testing.T) {
		rh2 := NewReactionHandler(wu, ac, &stubSessionAPI{msg: &discord.Message{
			Author: discord.User{ID: botID}, Embeds: []discord.Embed{{URL: "https://example.com"}},
		}, user: &discord.User{ID: botID}})
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("get auction err", func(t *testing.T) {
		rh2 := NewReactionHandler(wu, &stubListing{err: errors.New("a")}, sess)
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("register err", func(t *testing.T) {
		rrepo := &memWatchRepo{addErr: errors.New("a")}
		rh2 := NewReactionHandler(appwatch.NewWatchUsecase(rrepo), ac, sess)
		rh2.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
	t.Run("register ok", func(t *testing.T) {
		rh.HandleReactionAdd(&gateway.MessageReactionAddEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji:  discord.Emoji{Name: "\U0001F514"},
			GuildID: discord.GuildID(5),
		})
	})
	t.Run("remove flows", func(t *testing.T) {
		rh.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
		rh.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: botID, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
		rh2 := NewReactionHandler(wu, ac, &stubSessionAPI{msgErr: errors.New("f"), user: &discord.User{ID: botID}})
		rh2.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
		rh3 := NewReactionHandler(wu, ac, &stubSessionAPI{msg: &discord.Message{
			Author: discord.User{ID: 2}, Embeds: msg.Embeds,
		}, user: &discord.User{ID: botID}})
		rh3.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
		rh4 := NewReactionHandler(wu, ac, &stubSessionAPI{msg: &discord.Message{
			Author: discord.User{ID: botID}, Embeds: []discord.Embed{{URL: "https://example.com"}},
		}, user: &discord.User{ID: botID}})
		rh4.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
		rrepo := &memWatchRepo{remErr: errors.New("r")}
		rh5 := NewReactionHandler(appwatch.NewWatchUsecase(rrepo), ac, sess)
		rh5.HandleReactionRemove(&gateway.MessageReactionRemoveEvent{
			UserID: 2, ChannelID: ch, MessageID: mid,
			Emoji: discord.Emoji{Name: "\U0001F514"},
		})
	})
}
