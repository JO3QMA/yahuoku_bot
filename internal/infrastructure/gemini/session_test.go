package gemini

import (
	"context"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
)

func Test_extract_skips_search_when_resolved(t *testing.T) {
	stage1 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":[{"key":"model","value":"RTX 3080"}],"missing_keys":[],"candidate_queries":[]}`
	stage4 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":[{"key":"model","value":"RTX 3080"}]}`
	toolsCalled := false
	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	api.stubGenerate = func(_ context.Context, _ string, _ []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		if cfg != nil && len(cfg.Tools) > 0 {
			toolsCalled = true
		}
		text := stage4
		if cfg != nil && cfg.ResponseSchema != nil {
			if _, ok := cfg.ResponseSchema.Properties["missing_keys"]; ok {
				text = stage1
			}
		}
		return jsonResponse(text), nil
	}

	pd, err := NewTestClient(api, Options{}).Extract(context.Background(), product.ExtractInput{
		Title: "GPU", Description: "NVIDIA GeForce RTX 3080 10GB",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "gpu" {
		t.Fatalf("%v", pd)
	}
	if toolsCalled {
		t.Fatal("search supplement should not run when unresolved is empty")
	}
}

func Test_extract_text_only(t *testing.T) {
	stage1 := `{"category":"other","condition":"","shipping_free":null,"fields":[],"missing_keys":[],"candidate_queries":[]}`
	stage4 := `{"category":"other","condition":"","fields":[]}`
	call := 0
	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	api.stubGenerate = func(_ context.Context, _ string, _ []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		call++
		text := stage4
		if cfg != nil && cfg.ResponseSchema != nil {
			if _, ok := cfg.ResponseSchema.Properties["missing_keys"]; ok {
				text = stage1
			}
		}
		return jsonResponse(text), nil
	}

	pd, err := NewTestClient(api, Options{FastModel: "m"}).Extract(context.Background(), product.ExtractInput{Title: "t", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if pd == nil {
		t.Fatal("nil product")
	}
	if call < 2 {
		t.Fatalf("expected at least 2 api calls, got %d", call)
	}
}
