package gemini

import (
	"context"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
)

func Test_session_skips_search_when_resolved(t *testing.T) {
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

	pd, err := NewTestClient(api, Options{ExtractionMode: ExtractionModeSession}).Extract(context.Background(), product.ExtractInput{
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

func Test_client_session_mode(t *testing.T) {
	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	c := NewTestClient(api, Options{ExtractionMode: ExtractionModeSession})
	if c.(*client).sess == nil {
		t.Fatal("expected session client")
	}
	if c.(*client).p != nil {
		t.Fatal("pipeline should be nil in session mode")
	}
}

func Test_client_pipeline_default(t *testing.T) {
	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	c := NewTestClient(api, Options{})
	if c.(*client).p == nil {
		t.Fatal("expected pipeline client")
	}
	if c.(*client).sess != nil {
		t.Fatal("session should be nil in pipeline mode")
	}
}
