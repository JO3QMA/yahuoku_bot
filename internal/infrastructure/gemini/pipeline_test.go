package gemini

import (
	"context"
	"testing"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"google.golang.org/genai"
)

func Test_shouldRunStage3(t *testing.T) {
	if shouldRunStage3(&stage1Result{MissingKeys: []string{"cpu_model_line"}}) != true {
		t.Fatal("expected true")
	}
	if shouldRunStage3(&stage1Result{MissingKeys: nil}) != false {
		t.Fatal("expected false")
	}
}

func Test_pipeline_with_stage3_skipped(t *testing.T) {
	stage1 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":[{"key":"model","value":"RTX 3080"}],"missing_keys":[],"candidate_queries":[]}`
	stage4 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":[{"key":"model","value":"RTX 3080"}]}`
	generateHook = func(_ context.Context, _ *genai.Client, _ string, _ []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		text := stage4
		if cfg != nil && cfg.ResponseSchema != nil {
			if _, ok := cfg.ResponseSchema.Properties["missing_keys"]; ok {
				text = stage1
			}
		}
		return jsonResponse(text), nil
	}
	t.Cleanup(func() { generateHook = nil })

	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	pd, err := newPipeline(api, Options{}).run(context.Background(), appauction.ExtractInput{
		Title: "GPU", Description: "NVIDIA GeForce RTX 3080 10GB",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "gpu" {
		t.Fatalf("%v", pd)
	}
}

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
