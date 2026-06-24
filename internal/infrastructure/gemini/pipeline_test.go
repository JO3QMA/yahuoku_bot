package gemini

import (
	"context"
	"errors"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
)

func Test_executeLookupSpec_respects_limit(t *testing.T) {
	p := newPipeline(&genAIAPI{}, Options{MaxSearchCalls: 1}.Normalize())
	count := 1
	notes := []string{}
	fr := p.executeLookupSpec(context.Background(), &genai.FunctionCall{
		Name: "lookup_spec",
		Args: map[string]any{"query": "test", "field_key": "model"},
	}, &count, &notes)
	if fr["error"] != "search limit reached" {
		t.Fatalf("%v", fr)
	}
}

func Test_executeLookupSpec_does_not_increment_on_failure(t *testing.T) {
	api := &genAIAPI{}
	api.stubGroundedSearch = func(context.Context, string, string) (string, []string, error) {
		return "", nil, errors.New("search failed")
	}

	p := newPipeline(api, Options{MaxSearchCalls: 3}.Normalize())
	count := 0
	notes := []string{}
	fr := p.executeLookupSpec(context.Background(), &genai.FunctionCall{
		Name: "lookup_spec",
		Args: map[string]any{"query": "test", "field_key": "model"},
	}, &count, &notes)
	if fr["error"] == nil {
		t.Fatalf("expected error response, got %v", fr)
	}
	if count != 0 {
		t.Fatalf("expected count 0 on failure, got %d", count)
	}
}

func Test_pipeline_with_stage3_skipped(t *testing.T) {
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
		t.Fatal("stage3 tools should not be called")
	}
}

func Test_shouldRunStage3(t *testing.T) {
	if !shouldRunStage3(&stage1Result{MissingKeys: []string{"cpu_model_line"}}, nil) {
		t.Fatal("expected true")
	}
	if shouldRunStage3(&stage1Result{MissingKeys: nil}, nil) {
		t.Fatal("expected false")
	}
}

func Test_remainingMissingKeys_stage2FillsKey(t *testing.T) {
	s1 := &stage1Result{
		MissingKeys: []string{"cpu_model_line", "memory"},
		Fields:      []product.Field{{Key: "memory", Value: "32GB"}},
	}
	s2 := &stage2Result{
		ImageFields: []imageField{{Key: "cpu_model_line", Value: "Xeon E5", Confidence: "high"}},
	}
	keys := remainingMissingKeys(s1, s2)
	if len(keys) != 0 {
		t.Fatalf("expected no remaining keys, got %v", keys)
	}
	if shouldRunStage3(s1, s2) {
		t.Fatal("expected stage3 skipped when stage2 filled missing key")
	}
}

func Test_remainingMissingKeys_stage1FieldFillsKey(t *testing.T) {
	s1 := &stage1Result{
		MissingKeys: []string{"model"},
		Fields:      []product.Field{{Key: "model", Value: "RTX 3080"}},
	}
	if keys := remainingMissingKeys(s1, nil); len(keys) != 0 {
		t.Fatalf("got %v", keys)
	}
}
