package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"

	"github.com/jo3qma/sansai"
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

func (fakePreviewGem) Extract(context.Context, product.ExtractInput) (*product.Product, error) {
	return &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "x"}},
	}, nil
}

func stubPreviewSansai(t *testing.T, err error) {
	t.Helper()
	prev := applisting.SansaiGetItem
	applisting.SansaiGetItem = func(context.Context, sansai.Market, string) (*sansai.Item, error) {
		if err != nil {
			return nil, err
		}
		end := time.Now()
		return &sansai.Item{
			Market: sansai.MarketYahooAuction, ID: "id", Title: "T", Price: 1,
			Description: "d", SaleType: "auction",
			EndTime: end.Format(time.RFC3339), IsActive: true,
		}, nil
	}
	t.Cleanup(func() { applisting.SansaiGetItem = prev })
}

func TestRunPreview_executeErr(t *testing.T) {
	stubPreviewSansai(t, errors.New("x"))
	c := RunPreview(&bytes.Buffer{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_encodeErr(t *testing.T) {
	stubPreviewSansai(t, nil)
	c := RunPreview(errWriter{}, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) { return 0, errors.New("w") }

func TestRunPreview_emptyProductExit(t *testing.T) {
	stubPreviewSansai(t, nil)
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &emptyProductGem{}, nil
		},
	})
	if c != 1 {
		t.Fatalf("code=%d", c)
	}
}

type emptyProductGem struct{}

func (emptyProductGem) Extract(context.Context, product.ExtractInput) (*product.Product, error) {
	return &product.Product{}, nil
}

func TestRunPreview_success(t *testing.T) {
	stubPreviewSansai(t, nil)
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"yahoo_auction", "id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
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
