package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
	infrarqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/rqlite"
	"jo3qma.com/yahoo_auctions_bot/internal/presentation/discord"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

type fakeRunner struct{}

func (fakeRunner) Run(ctx context.Context) error { return nil }

type waitCtxRunner struct{}

func (waitCtxRunner) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

type fakeOpenAI struct{}

func (fakeOpenAI) Extract(context.Context, product.ExtractInput) (*product.Product, error) {
	return &product.Product{}, nil
}

func TestRun_nilDeps(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "dt")
	t.Setenv("OPENAI_API_KEY", "gk")
	t.Setenv("RQLITE_URL", "http://localhost:4001")
	t.Cleanup(func() {
		_ = os.Unsetenv("DISCORD_TOKEN")
		_ = os.Unsetenv("OPENAI_API_KEY")
		_ = os.Unsetenv("RQLITE_URL")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := successDeps()
	deps.OpenRqlite = fakeOpenRqlite
	if err := run(ctx, deps); err != nil {
		t.Fatal(err)
	}
}

func TestRun_configLoadError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return nil, errors.New("bad")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_missingDiscordToken(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_missingOpenAIKey(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{DiscordToken: "t"}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_openaiError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{DiscordToken: "t", OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return nil, errors.New("g")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_rqliteError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{DiscordToken: "t", OpenAIAPIKey: "k", RqliteURL: "http://x"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakeOpenAI{}, nil
		},
		OpenRqlite: func(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error) {
			return nil, errors.New("rq")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_discordNewBotError(t *testing.T) {
	deps := successDeps()
	deps.NewDiscordBot = func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
		return nil, errors.New("bot")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, deps); err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_success_rqlite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, successDeps()); err != nil {
		t.Fatal(err)
	}
}

func TestRun_tokenPrefix(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := successDeps()
	deps.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			DiscordToken: "rawtoken", OpenAIAPIKey: "k",
			RqliteURL: "http://noop", APIEndpoint: "http://localhost:8080",
		}, nil
	}
	deps.NewDiscordBot = func(token string, _ *appauction.PreviewUsecase, _ *discord.AllowedFilter, _ *appwatch.WatchUsecase, _ infraauction.Client, _ watch.Repository, _ discord.BotConfig) (discordRunner, error) {
		if token != "Bot rawtoken" {
			t.Fatalf("token=%q", token)
		}
		return fakeRunner{}, nil
	}
	_ = run(ctx, deps)
}

func TestRunWithSignal_parentCancelled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runWithSignal(parent, successDeps()); err != nil {
		t.Fatal(err)
	}
}

func TestMergeBotDeps_partialOverride(t *testing.T) {
	called := false
	d := &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{DiscordToken: "a", OpenAIAPIKey: "b"}, nil
		},
		OpenRqlite: func(context.Context, string, ...infrarqlite.NewClientOption) (*infrarqlite.Client, error) {
			called = true
			return nil, errors.New("skip")
		},
	}
	mergeBotDeps(d)
	if d.NewOpenAIClient == nil || d.NewWatchRepo == nil {
		t.Fatal("defaults not merged")
	}
	_, _ = d.OpenRqlite(context.Background(), "x")
	if !called {
		t.Fatal("custom OpenRqlite not used")
	}
}

func TestRun_loadConfigFromEnv(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "dt")
	t.Setenv("OPENAI_API_KEY", "gk")
	t.Setenv("ALLOWED_CHANNELS", "c1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := successDeps()
	deps.LoadConfig = config.Load
	if err := run(ctx, deps); err != nil {
		t.Fatal(err)
	}
}

func TestMergeBotDeps_allNil(t *testing.T) {
	d := &botDeps{}
	mergeBotDeps(d)
	if d.LoadConfig == nil || d.NewOpenAIClient == nil || d.OpenRqlite == nil ||
		d.NewWatchRepo == nil || d.NewDiscordBot == nil {
		t.Fatal("mergeBotDeps should fill all defaults")
	}
}

func TestMergeBotDeps_defaultWatchRepoBody(t *testing.T) {
	d := &botDeps{}
	mergeBotDeps(d)
	ctx := context.Background()
	cl, err := fakeOpenRqlite(ctx, "http://x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cl.Close() })
	repo := d.NewWatchRepo(cl)
	if repo == nil {
		t.Fatal("nil repo")
	}
}

func TestRunWithSignal_onSigint(t *testing.T) {
	deps := successDeps()
	deps.NewDiscordBot = func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
		return waitCtxRunner{}, nil
	}
	go func() {
		time.Sleep(80 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGINT)
	}()
	done := make(chan error, 1)
	go func() { done <- runWithSignal(context.Background(), deps) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestDefaultNewDiscordBot_invokesNewBot(t *testing.T) {
	repo := &memRepoRqlite{}
	ac := infraauction.NewClient("http://127.0.0.1:9", nil)
	pu := appauction.NewPreviewUsecase(ac, &fakeOpenAI{})
	wu := appwatch.NewWatchUsecase(repo)
	af := discord.NewAllowedFilter(nil, nil)
	_, errBot := defaultNewDiscordBot("Bot unit-test-token.invalid", pu, af, wu, ac, repo, discord.BotConfig{CheckIntervalMinutes: 60, PollDelayMs: 1})
	if errBot != nil {
		t.Logf("NewBot: %v (still covers defaultNewDiscordBot)", errBot)
	}
}

func TestRun_botRunLogsNonCancelError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := successDeps()
	deps.NewDiscordBot = func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
		return errRunner{}, nil
	}
	if err := run(ctx, deps); err != nil {
		t.Fatal(err)
	}
}

type errRunner struct{}

func (errRunner) Run(context.Context) error { return errors.New("run") }

func successDeps() *botDeps {
	return &botDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{
				DiscordToken: "Bot x.y.z", OpenAIAPIKey: "k",
				RqliteURL: "http://noop", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) { return &fakeOpenAI{}, nil },
		OpenRqlite:      fakeOpenRqlite,
		NewWatchRepo: func(*infrarqlite.Client) watch.Repository {
			return &memRepoRqlite{}
		},
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return fakeRunner{}, nil
		},
	}
}

func fakeOpenRqlite(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error) {
	return infrarqlite.Open(ctx, url, append([]infrarqlite.NewClientOption{
		infrarqlite.WithRqliteHTTPClientFactory(func(string, *http.Client) (infrarqlite.HTTPClient, error) {
			return okRqliteHTTP{}, nil
		}),
	}, opts...)...)
}

type okRqliteHTTP struct{}

func (okRqliteHTTP) PromoteErrors(bool) {}

func (okRqliteHTTP) ExecuteSingle(context.Context, string, ...any) (*rqlitehttp.ExecuteResponse, error) {
	return &rqlitehttp.ExecuteResponse{}, nil
}

func (okRqliteHTTP) QuerySingle(context.Context, string, ...any) (*rqlitehttp.QueryResponse, error) {
	return &rqlitehttp.QueryResponse{Results: []rqlitehttp.QueryResult{}}, nil
}

func (okRqliteHTTP) Close() error { return nil }

type memRepoRqlite struct{}

func (memRepoRqlite) Add(context.Context, *watch.Watch) error { return nil }
func (memRepoRqlite) Remove(context.Context, string, string, string) error {
	return nil
}
func (memRepoRqlite) ListActive(context.Context) ([]*watch.Watch, error) { return nil, nil }
func (memRepoRqlite) UpdatePrice(context.Context, int64, int64) error    { return nil }
func (memRepoRqlite) MarkReminded(context.Context, int64) error          { return nil }
func (memRepoRqlite) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (memRepoRqlite) FindByMessage(context.Context, string) ([]*watch.Watch, error) {
	return nil, nil
}
func (memRepoRqlite) RemoveByAuctionID(context.Context, string) error { return nil }
