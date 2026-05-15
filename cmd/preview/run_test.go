package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

func TestRunPreview_usage(t *testing.T) {
	if c := RunPreview(&bytes.Buffer{}, nil, "x", nil); c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_configErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, "nope.yaml", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return nil, errors.New("e")
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_noOpenAIKey(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{}, nil
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

func TestRunPreview_openAIClientErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k"}, nil
		},
		NewSpecExtractor: func(string, string, string) (appauction.SpecExtractor, error) {
			return nil, errors.New("gc")
		},
	})
	if c != 2 {
		t.Fatalf("code=%d", c)
	}
}

type fakePreviewGem struct{}

func (fakePreviewGem) ExtractSpec(context.Context, string, string) (*spec.Spec, error) {
	return &spec.Spec{CPUModelLine: "x"}, nil
}

func TestRunPreview_executeErr(t *testing.T) {
	c := RunPreview(&bytes.Buffer{}, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewSpecExtractor: func(string, string, string) (appauction.SpecExtractor, error) {
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
	c := RunPreview(errWriter{}, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewSpecExtractor: func(string, string, string) (appauction.SpecExtractor, error) {
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

func TestRunPreview_emptySpecExit(t *testing.T) {
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewSpecExtractor: func(string, string, string) (appauction.SpecExtractor, error) {
			return &emptySpecGem{}, nil
		},
		NewAuctionClient: func(string) infraauction.Client {
			return &okAuction{}
		},
	})
	if c != 1 {
		t.Fatalf("code=%d", c)
	}
}

type emptySpecGem struct{}

func (emptySpecGem) ExtractSpec(context.Context, string, string) (*spec.Spec, error) {
	return &spec.Spec{}, nil
}

func TestRunPreview_success(t *testing.T) {
	var buf bytes.Buffer
	c := RunPreview(&buf, []string{"id"}, "x", &previewDeps{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{OpenAIAPIKey: "k", APIEndpoint: "http://localhost:8080"}, nil
		},
		NewSpecExtractor: func(string, string, string) (appauction.SpecExtractor, error) {
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

func TestIsSpecEmpty(t *testing.T) {
	if !isSpecEmpty(nil) {
		t.Fatal()
	}
	if !isSpecEmpty(&spec.Spec{}) {
		t.Fatal()
	}
	if isSpecEmpty(&spec.Spec{Condition: "中古"}) {
		t.Fatal()
	}
}
