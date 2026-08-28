package rqlite

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

func TestSplitSchema(t *testing.T) {
	out := splitSchema(" A ; ; B ")
	if len(out) != 2 || out[0] != "A" || out[1] != "B" {
		t.Fatalf("%#v", out)
	}
}

func TestMigrateSchemaIfNeeded_skipsNewSchema(t *testing.T) {
	f := &fakeHTTP{queryResp: &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{Values: [][]any{{"market"}, {"listing_id"}}}},
	}}
	if err := migrateSchemaIfNeeded(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if len(f.executeStatements) != 0 {
		t.Fatalf("unexpected drop: %#v", f.executeStatements)
	}
}

func TestMigrateSchemaIfNeeded_dropsLegacySchema(t *testing.T) {
	f := &fakeHTTP{queryResp: &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{Values: [][]any{{"auction_id"}}}},
	}}
	if err := migrateSchemaIfNeeded(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if len(f.executeStatements) != 1 || f.executeStatements[0] != dropWatchItems {
		t.Fatalf("got %#v", f.executeStatements)
	}
}

func TestMigrateSchemaIfNeeded_freshDB(t *testing.T) {
	f := &fakeHTTP{}
	if err := migrateSchemaIfNeeded(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if len(f.executeStatements) != 0 {
		t.Fatalf("unexpected execute: %#v", f.executeStatements)
	}
}

func TestIsRetryableRqliteError(t *testing.T) {
	if !isRetryableRqliteError(errors.New("503 Service")) {
		t.Fatal("503")
	}
	if !isRetryableRqliteError(errors.New("leader not found x")) {
		t.Fatal("leader")
	}
	if isRetryableRqliteError(errors.New("other")) {
		t.Fatal("not retryable")
	}
}

type fakeHTTP struct {
	promoted          bool
	execCalls         int
	execErrs          []error
	queryResp         *rqlitehttp.QueryResponse
	queryErr          error
	closed            bool
	executeStatements []string
}

func (f *fakeHTTP) PromoteErrors(b bool) { f.promoted = b }

func (f *fakeHTTP) ExecuteSingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.ExecuteResponse, error) {
	f.executeStatements = append(f.executeStatements, statement)
	if f.execCalls < len(f.execErrs) {
		err := f.execErrs[f.execCalls]
		f.execCalls++
		return nil, err
	}
	f.execCalls++
	return &rqlitehttp.ExecuteResponse{}, nil
}

func (f *fakeHTTP) QuerySingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.QueryResponse, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.queryResp, nil
}

func (f *fakeHTTP) Close() error {
	f.closed = true
	return nil
}

func TestOpen_clientFactoryError(t *testing.T) {
	_, err := Open(context.Background(), "u", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return nil, errors.New("nf")
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_schemaRetryThenSuccess(t *testing.T) {
	f := &fakeHTTP{execErrs: []error{errors.New("503")}}
	_, err := Open(context.Background(), "http://x", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return f, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !f.promoted {
		t.Fatal("promote")
	}
}

func TestOpen_secondSchemaStatementFails(t *testing.T) {
	f := &fakeHTTP{execErrs: []error{nil, errors.New("syntax error")}}
	_, err := Open(context.Background(), "http://x", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return f, nil
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_schemaNonRetryable(t *testing.T) {
	f := &fakeHTTP{execErrs: []error{errors.New("syntax")}}
	_, err := Open(context.Background(), "http://x", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return f, nil
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if !f.closed {
		t.Fatal("client should be closed on failure")
	}
}

type fake503 struct{}

func (fake503) PromoteErrors(bool) {}
func (fake503) ExecuteSingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.ExecuteResponse, error) {
	return nil, errors.New("503")
}
func (fake503) QuerySingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.QueryResponse, error) {
	return nil, errors.New("unused")
}
func (fake503) Close() error { return nil }

func TestOpen_schemaExhaustRetries(t *testing.T) {
	af := &fake503{}
	_, err := Open(context.Background(), "http://x", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return af, nil
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_ctxCancelDuringBackoff(t *testing.T) {
	f := &fakeHTTP{execErrs: []error{errors.New("503")}}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()
	_, err := Open(ctx, "http://x", WithRqliteHTTPClientFactory(func(string, *http.Client) (HTTPClient, error) {
		return f, nil
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestClient_Close(t *testing.T) {
	f := &fakeHTTP{}
	c := &Client{h: f}
	_ = c.Close()
	if !f.closed {
		t.Fatal("close not forwarded")
	}
}

func TestParseQueryResults_topLevelError(t *testing.T) {
	qr := &rqlitehttp.QueryResponse{Error: "boom"}
	_, err := parseQueryResults(qr)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseQueryResults_resultError(t *testing.T) {
	qr := &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{Error: "e"}},
	}
	_, err := parseQueryResults(qr)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseQueryResults_empty(t *testing.T) {
	qr := &rqlitehttp.QueryResponse{Results: []rqlitehttp.QueryResult{}}
	items, err := parseQueryResults(qr)
	if err != nil || items != nil {
		t.Fatalf("%v %#v", err, items)
	}
}

func TestRowToWatch_shortRow(t *testing.T) {
	_, err := rowToWatch([]any{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRowToWatch_types(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Second)
	row := []any{
		json.Number("1"), "yahoo_auction", "a", "u", "g", "c", "m",
		json.Number("99"),
		created.Format(time.RFC3339),
		json.Number("0"),
		"",
		created.Format(time.RFC3339),
	}
	item, err := rowToWatch(row)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != 1 || item.LastKnownPrice != 99 || item.Reminded {
		t.Fatalf("%+v", item)
	}
	row[7] = float64(100)
	row[9] = int64(1)
	item2, err := rowToWatch(row)
	if err != nil || !item2.Reminded {
		t.Fatal(item2, err)
	}
}

func TestToInt64_bad(t *testing.T) {
	_, err := toInt64("x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToString_nil(t *testing.T) {
	if toString(nil) != "" {
		t.Fatal()
	}
	if toString(123) != "123" {
		t.Fatal()
	}
}

func TestParseTime_invalid(t *testing.T) {
	if parseTime("not-a-time") != nil {
		t.Fatal()
	}
}

func TestWatchRepository_Add_nilEndTime(t *testing.T) {
	f := &fakeHTTP{}
	repo := NewWatchRepository(&Client{h: f})
	ctx := context.Background()
	err := repo.Add(ctx, &watch.Watch{
		Market: listing.MarketYahooAuction, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1,
		EndTime:        nil,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatchRepository_methods(t *testing.T) {
	f := &fakeHTTP{}
	repo := NewWatchRepository(&Client{h: f})
	ctx := context.Background()
	_ = repo.Remove(ctx, listing.MarketYahooAuction, "a", "u", "m")
	_ = repo.RemoveByListing(ctx, listing.MarketYahooAuction, "a")
	_ = repo.UpdatePrice(ctx, 1, 2)
	_ = repo.MarkReminded(ctx, 1)
	_ = repo.UpdateThreadID(ctx, "m", "t")
}

func TestWatchRepository_List_queryErr(t *testing.T) {
	f := &fakeHTTP{queryErr: errors.New("q")}
	repo := NewWatchRepository(&Client{h: f})
	_, err := repo.ListActive(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_Find_queryErr(t *testing.T) {
	f := &fakeHTTP{queryErr: errors.New("q")}
	repo := NewWatchRepository(&Client{h: f})
	_, err := repo.FindByMessage(context.Background(), "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_List_parseRows(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Second)
	qr := &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{
			Values: [][]any{{
				int64(5), "yahoo_auction", "aid", "uid", "gid", "cid", "mid",
				int64(1000),
				nil,
				int64(0),
				"",
				created.Format(time.RFC3339),
			}},
		}},
	}
	f := &fakeHTTP{queryResp: qr}
	repo := NewWatchRepository(&Client{h: f})
	items, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ListingID != "aid" {
		t.Fatalf("%+v", items[0])
	}
}

type execFailHTTP struct{}

func (execFailHTTP) PromoteErrors(bool) {}

func (execFailHTTP) ExecuteSingle(context.Context, string, ...any) (*rqlitehttp.ExecuteResponse, error) {
	return nil, errors.New("exec")
}

func (execFailHTTP) QuerySingle(context.Context, string, ...any) (*rqlitehttp.QueryResponse, error) {
	return nil, errors.New("query")
}

func (execFailHTTP) Close() error { return nil }

func TestWatchRepository_executeFailures(t *testing.T) {
	repo := NewWatchRepository(&Client{h: execFailHTTP{}})
	ctx := context.Background()
	_ = repo.Add(ctx, &watch.Watch{Market: listing.MarketYahooAuction, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m", LastKnownPrice: 1})
	_ = repo.Remove(ctx, listing.MarketYahooAuction, "a", "u", "m")
	_ = repo.RemoveByListing(ctx, listing.MarketYahooAuction, "a")
	_ = repo.UpdatePrice(ctx, 1, 2)
	_ = repo.MarkReminded(ctx, 1)
	_ = repo.UpdateThreadID(ctx, "m", "t")
}

func TestParseQueryResults_badRow(t *testing.T) {
	qr := &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{
			Values: [][]any{{int64(1), "a"}},
		}},
	}
	_, err := parseQueryResults(qr)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_FindByMessage_ok(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Second)
	qr := &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{
			Values: [][]any{{
				int64(5), "yahoo_auction", "aid", "uid", "gid", "cid", "mid",
				int64(1000),
				nil,
				int64(0),
				"",
				created.Format(time.RFC3339),
			}},
		}},
	}
	f := &fakeHTTP{queryResp: qr}
	repo := NewWatchRepository(&Client{h: f})
	items, err := repo.FindByMessage(context.Background(), "mid")
	if err != nil || len(items) != 1 {
		t.Fatalf("%v %#v", err, items)
	}
}

func TestWatchRepository_Add_withEndTime(t *testing.T) {
	f := &fakeHTTP{}
	repo := NewWatchRepository(&Client{h: f})
	et := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()
	err := repo.Add(ctx, &watch.Watch{
		Market: listing.MarketYahooAuction, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1,
		EndTime:        &et,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatchRepository_Add_onConflictResetsReminded(t *testing.T) {
	f := &fakeHTTP{}
	repo := NewWatchRepository(&Client{h: f})
	ctx := context.Background()
	item := &watch.Watch{
		Market: listing.MarketYahooAuction, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1000,
	}
	if err := repo.Add(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := repo.Add(ctx, &watch.Watch{
		Market: listing.MarketYahooAuction, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 2000,
	}); err != nil {
		t.Fatal(err)
	}
	if len(f.executeStatements) < 2 {
		t.Fatalf("expected 2 execute calls, got %d", len(f.executeStatements))
	}
	if !strings.Contains(f.executeStatements[1], "reminded = 0") {
		t.Fatalf("expected reminded reset in upsert SQL, got %q", f.executeStatements[1])
	}
}

func TestParseTime_emptyString(t *testing.T) {
	if parseTime("") != nil {
		t.Fatal("expected nil")
	}
}

func TestParseQueryResults_errorViaJSONResults(t *testing.T) {
	raw := `{"results":[{"error":"stmt failed"}]}`
	var qr rqlitehttp.QueryResponse
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		t.Fatal(err)
	}
	_, err := parseQueryResults(&qr)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseQueryResults_emptyResultsFromJSON(t *testing.T) {
	raw := `{"results":[]}`
	var qr rqlitehttp.QueryResponse
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		t.Fatal(err)
	}
	items, err := parseQueryResults(&qr)
	if err != nil || items != nil {
		t.Fatalf("%v %#v", err, items)
	}
}

func TestParseQueryResults_emptyValuesInOneResult(t *testing.T) {
	qr := &rqlitehttp.QueryResponse{
		Results: []rqlitehttp.QueryResult{{Values: [][]any{}, Error: ""}},
	}
	items, err := parseQueryResults(qr)
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %#v", err, items)
	}
}

func TestOpen_usesDefaultRqliteHTTPClient(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Open(ctx, "http://127.0.0.1:19999")
	if err == nil {
		t.Fatal("expected error connecting")
	}
}
