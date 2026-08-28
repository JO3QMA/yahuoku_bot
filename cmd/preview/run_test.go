package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/config"
	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	infralisting "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

func TestRunPreview_usage(t *testing.T) {
	if c := RunPreview(&bytes.Buffer{}, nil, nil); c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_unknownMarket(t *testing.T) {
	if c := RunPreview(&bytes.Buffer{}, []string{"ebay", "id"}, nil); c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_configErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return nil, errors.New("e")
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_noOpenAI(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_openaiClientErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return nil, errors.New("gc")
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type fakePreviewGem struct{}

func (fakePreviewGem) Extract(context.Context, string, string, []string) (*product.Product, error) {
	return &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "x"}},
	}, nil
}

func TestRunPreview_executeErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewListingClient: func() infralisting.Client { return &failListing{} },
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type failListing struct{}

func (failListing) Get(context.Context, dlisting.Ref) (*dlisting.Data, error) {
	return nil, errors.New("x")
}

func TestRunPreview_encodeErr(t *testing.T) {
	c := RunPreview(errWriter{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewListingClient: func() infralisting.Client { return &okListing{} },
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) { return 0, errors.New("w") }

type okListing struct{}

func (okListing) Get(context.Context, dlisting.Ref) (*dlisting.Data, error) {
	end := time.Now()
	return &dlisting.Data{
		Ref:         dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: "id"},
		Title:       "T",
		Price:       1,
		Description: "d",
		EndTime:     &end,
		SaleType:    dlisting.SaleTypeAuction,
		IsActive:    true,
	}, nil
}

func TestRunPreview_emptyProductExit(t *testing.T) {
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &emptyProductGem{}, nil
		},
		NewListingClient: func() infralisting.Client { return &okListing{} },
	})
	if c != 1 {
		t.Fatalf("code=%d", c)
	}
}

type emptyProductGem struct{}

func (emptyProductGem) Extract(context.Context, string, string, []string) (*product.Product, error) {
	return &product.Product{}, nil
}

func TestRunPreview_success(t *testing.T) {
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewListingClient: func() infralisting.Client { return &okListing{} },
	})
	if c != 0 {
		t.Fatalf("code=%d", c)
	}
	var v map[string]any
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
}

func TestProduct_IsEffectivelyEmpty_cli(t *testing.T) {
	var nilPD *product.Product
	if !nilPD.IsEffectivelyEmpty() {
		t.Fatal("nil")
	}
	if !product.EmptyProduct().IsEffectivelyEmpty() {
		t.Fatal("empty")
	}
	p := &product.Product{Condition: "中古"}
	if p.IsEffectivelyEmpty() {
		t.Fatal("has condition")
	}
}
