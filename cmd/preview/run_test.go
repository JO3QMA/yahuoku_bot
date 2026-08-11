package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

func TestRunPreview_usage(t *testing.T) {
	if c := RunPreview(&bytes.Buffer{}, nil, nil); c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_configErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return nil, errors.New("e")
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_noOpenAI(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_openaiClientErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, &previewDeps{
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

func TestRunPreview_executeErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewAuctionClient: func(string) infraauction.Client {
			return &failAuction{}
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type failAuction struct{}

func (failAuction) GetAuction(context.Context, string) (*infraauction.AuctionData, error) {
	return nil, errors.New("x")
}

func TestRunPreview_encodeErr(t *testing.T) {
	c := RunPreview(errWriter{}, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewAuctionClient: func(string) infraauction.Client {
			return &okAuction{}
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) { return 0, errors.New("w") }

type okAuction struct{}

func (okAuction) GetAuction(context.Context, string) (*infraauction.AuctionData, error) {
	end := time.Now()
	return &infraauction.AuctionData{
		AuctionID: "a", Title: "T", CurrentPrice: 1, Status: "S", Description: "d", EndTime: &end,
	}, nil
}

func TestRunPreview_emptyProductExit(t *testing.T) {
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &emptyProductGem{}, nil
		},
		NewAuctionClient: func(string) infraauction.Client {
			return &okAuction{}
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
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"id"}, &previewDeps{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewOpenAIClient: func(cfg *config.Config) (openai.Client, error) {
			return &fakePreviewGem{}, nil
		},
		NewAuctionClient: func(string) infraauction.Client {
			return &okAuction{}
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
