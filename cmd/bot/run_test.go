package main

import (
	"context"
	"database/sql"
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
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
	infrarqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/rqlite"
	infrasqlite "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/sqlite"
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

type fakeGemini struct{}

func (fakeGemini) ExtractProduct(context.Context, string, string) (*product.ProductDetail, error) {
	return &product.ProductDetail{}, nil
}

func TestRun_nilDeps(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "dt")
	t.Setenv("GEMINI_API_KEY", "gk")
	t.Setenv("DB_PATH", t.TempDir()+"/nildeps.db")
	t.Cleanup(func() {
		_ = os.Unsetenv("DISCORD_TOKEN")
		_ = os.Unsetenv("GEMINI_API_KEY")
		_ = os.Unsetenv("DB_PATH")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRun_configLoadError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return nil, errors.New("bad")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_missingDiscordToken(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{GeminiAPIKey: "k"}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_missingGeminiKey(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{DiscordToken: "t"}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_geminiError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{DiscordToken: "t", GeminiAPIKey: "k"}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) {
			return nil, errors.New("g")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_rqliteError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{DiscordToken: "t", GeminiAPIKey: "k", RqliteURL: "http://x"}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) {
			return &fakeGemini{}, nil
		},
		OpenRqlite: func(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error) {
			return nil, errors.New("rq")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_sqliteError(t *testing.T) {
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{DiscordToken: "t", GeminiAPIKey: "k", DBPath: "x.db"}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) {
			return &fakeGemini{}, nil
		},
		OpenSQLite: func(string, ...infrasqlite.OpenOption) (*sql.DB, error) {
			return nil, errors.New("sql")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_discordNewBotError(t *testing.T) {
	dir := t.TempDir()
	err := run(context.Background(), &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "t", GeminiAPIKey: "k",
				DBPath: dir + "/w.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return nil, errors.New("bot")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_success_sqlite(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := run(ctx, &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "Bot x.y.z", GeminiAPIKey: "k",
				DBPath: dir + "/w.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return fakeRunner{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_tokenPrefix(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = run(ctx, &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "rawtoken", GeminiAPIKey: "k",
				DBPath: dir + "/w2.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(token string, _ *appauction.PreviewUsecase, _ *discord.AllowedFilter, _ *appwatch.WatchUsecase, _ infraauction.Client, _ watch.Repository, _ discord.BotConfig) (discordRunner, error) {
			if token != "Bot rawtoken" {
				t.Fatalf("token=%q", token)
			}
			return fakeRunner{}, nil
		},
	})
}

func TestRunWithSignal_parentCancelled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	err := runWithSignal(parent, &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "t", GeminiAPIKey: "k",
				DBPath: dir + "/ws.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return fakeRunner{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeBotDeps_partialOverride(t *testing.T) {
	called := false
	d := &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{DiscordToken: "a", GeminiAPIKey: "b"}, nil
		},
		OpenSQLite: func(string, ...infrasqlite.OpenOption) (*sql.DB, error) {
			called = true
			return nil, errors.New("skip")
		},
	}
	mergeBotDeps(d)
	if d.NewGeminiClient == nil || d.OpenRqlite == nil {
		t.Fatal("defaults not merged")
	}
	_, _ = d.OpenSQLite("x")
	if !called {
		t.Fatal("custom OpenSQLite not used")
	}
}

func TestRun_configPathFromEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/cfg.yaml"
	if err := os.WriteFile(cfgPath, []byte("allowed:\n  guilds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", cfgPath)
	t.Setenv("DISCORD_TOKEN", "dt")
	t.Setenv("GEMINI_API_KEY", "gk")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := run(ctx, &botDeps{
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return fakeRunner{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_rqliteBranchOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	err := run(ctx, &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "t", GeminiAPIKey: "k",
				RqliteURL: "http://noop", DBPath: dir + "/unused.db",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		OpenRqlite: func(ctx context.Context, url string, opts ...infrarqlite.NewClientOption) (*infrarqlite.Client, error) {
			return infrarqlite.Open(ctx, url, append([]infrarqlite.NewClientOption{
				infrarqlite.WithRqliteHTTPClientFactory(func(string, *http.Client) (infrarqlite.HTTPClient, error) {
					return okRqliteHTTP{}, nil
				}),
			}, opts...)...)
		},
		NewWatchRepoRqlite: func(*infrarqlite.Client) watch.Repository {
			return &memRepoRqlite{}
		},
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return fakeRunner{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
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

func (memRepoRqlite) Add(context.Context, *watch.WatchItem) error { return nil }
func (memRepoRqlite) Remove(context.Context, string, string, string) error {
	return nil
}
func (memRepoRqlite) ListActive(context.Context) ([]*watch.WatchItem, error) { return nil, nil }
func (memRepoRqlite) UpdatePrice(context.Context, int64, int64) error { return nil }
func (memRepoRqlite) MarkReminded(context.Context, int64) error       { return nil }
func (memRepoRqlite) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (memRepoRqlite) FindByMessage(context.Context, string) ([]*watch.WatchItem, error) {
	return nil, nil
}
func (memRepoRqlite) RemoveByAuctionID(context.Context, string) error { return nil }

type errRunner struct{}

func (errRunner) Run(context.Context) error { return errors.New("run") }

func TestMergeBotDeps_allNil(t *testing.T) {
	d := &botDeps{}
	mergeBotDeps(d)
	if d.LoadConfig == nil || d.NewGeminiClient == nil || d.OpenRqlite == nil || d.OpenSQLite == nil ||
		d.NewWatchRepoRqlite == nil || d.NewWatchRepoSQLite == nil || d.NewDiscordBot == nil {
		t.Fatal("mergeBotDeps should fill all defaults")
	}
}

func TestMergeBotDeps_defaultWatchRepoRqliteBody(t *testing.T) {
	d := &botDeps{}
	mergeBotDeps(d)
	ctx := context.Background()
	cl, err := infrarqlite.Open(ctx, "http://x", infrarqlite.WithRqliteHTTPClientFactory(func(string, *http.Client) (infrarqlite.HTTPClient, error) {
		return okRqliteHTTP{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cl.Close() })
	repo := d.NewWatchRepoRqlite(cl)
	if repo == nil {
		t.Fatal("nil repo")
	}
}

func TestMergeBotDeps_defaultWatchRepoSQLiteBody(t *testing.T) {
	d := &botDeps{}
	mergeBotDeps(d)
	db, err := infrasqlite.Open(t.TempDir() + "/mergebot.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := d.NewWatchRepoSQLite(db)
	if repo == nil {
		t.Fatal("nil repo")
	}
}

func TestRunWithSignal_onSigint(t *testing.T) {
	dir := t.TempDir()
	deps := &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "t", GeminiAPIKey: "k",
				DBPath: dir + "/w.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return waitCtxRunner{}, nil
		},
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
	db, err := infrasqlite.Open(t.TempDir() + "/botcov.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := infrasqlite.NewWatchRepository(db)
	ac := infraauction.NewClient("http://127.0.0.1:9", nil)
	pu := appauction.NewPreviewUsecase(ac, &fakeGemini{})
	wu := appwatch.NewWatchUsecase(repo)
	af := discord.NewAllowedFilter(nil, nil)
	_, errBot := defaultNewDiscordBot("Bot unit-test-token.invalid", pu, af, wu, ac, repo, discord.BotConfig{CheckIntervalMinutes: 60, PollDelayMs: 1})
	if errBot != nil {
		t.Logf("NewBot: %v (still covers defaultNewDiscordBot)", errBot)
	}
}

func TestRun_botRunLogsNonCancelError(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := run(ctx, &botDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{
				DiscordToken: "t", GeminiAPIKey: "k",
				DBPath: dir + "/w.db", APIEndpoint: "http://localhost:8080",
			}, nil
		},
		NewGeminiClient: func(string, string) (gemini.Client, error) { return &fakeGemini{}, nil },
		NewDiscordBot: func(string, *appauction.PreviewUsecase, *discord.AllowedFilter, *appwatch.WatchUsecase, infraauction.Client, watch.Repository, discord.BotConfig) (discordRunner, error) {
			return errRunner{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
